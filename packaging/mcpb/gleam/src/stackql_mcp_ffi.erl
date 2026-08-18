%% Erlang FFI for stackql_mcp: host facts (OS family, machine architecture,
%% home directory), the effectful acquisition primitives (file IO, HTTPS
%% download, sha256, .mcpb extraction) and the stdio relay used by the
%% launcher. The Gleam library core stays pure and takes host facts as
%% parameters so it is unit-testable without touching the host.
-module(stackql_mcp_ffi).
-export([os_family/0, arch/0, home/0, getenv/1, plain_arguments/0,
         file_exists/1, read_file/1, download/2, sha256_hex/1,
         extract_bundle/2, relay/2]).

%% "linux" | "darwin" | "windows"
os_family() ->
    case os:type() of
        {unix, darwin} -> <<"darwin">>;
        {unix, _}      -> <<"linux">>;
        {win32, _}     -> <<"windows">>
    end.

%% Best-effort machine architecture string, lowercased.
%% e.g. "x86_64-pc-linux-gnu" -> "x86_64"; "aarch64-..." -> "aarch64".
arch() ->
    Charlist = erlang:system_info(system_architecture),
    Full = unicode:characters_to_binary(Charlist),
    case binary:split(Full, <<"-">>) of
        [Head | _] -> string:lowercase(Head);
        []         -> string:lowercase(Full)
    end.

%% Home directory: HOME first (unix and most CI), then USERPROFILE (Windows).
home() ->
    case os:getenv("HOME") of
        false ->
            case os:getenv("USERPROFILE") of
                false -> <<"">>;
                Up    -> unicode:characters_to_binary(Up)
            end;
        H -> unicode:characters_to_binary(H)
    end.

%% Environment variable as a Gleam Result(String, Nil).
getenv(Name) ->
    case os:getenv(unicode:characters_to_list(Name)) of
        false -> {error, nil};
        Value -> {ok, unicode:characters_to_binary(Value)}
    end.

%% Program arguments after `gleam run -m <module> --`.
plain_arguments() ->
    [unicode:characters_to_binary(A) || A <- init:get_plain_arguments()].

file_exists(Path) ->
    filelib:is_regular(Path).

read_file(Path) ->
    case file:read_file(Path) of
        {ok, Bin} -> {ok, Bin};
        {error, Reason} -> {error, iolist_to_binary(io_lib:format("~s: ~p", [Path, Reason]))}
    end.

%% GET Url over HTTPS with certificate verification, following redirects
%% (the releases.stackql.io front door redirects to the release asset).
download(Url, UserAgent) ->
    {ok, _} = application:ensure_all_started(inets),
    {ok, _} = application:ensure_all_started(ssl),
    SslOpts = [{verify, verify_peer},
               {cacerts, public_key:cacerts_get()},
               {depth, 3},
               {customize_hostname_check,
                [{match_fun, public_key:pkix_verify_hostname_match_fun(https)}]}],
    Headers = [{"user-agent", unicode:characters_to_list(UserAgent)}],
    Request = {unicode:characters_to_list(Url), Headers},
    HttpOpts = [{ssl, SslOpts}, {timeout, 600000}, {connect_timeout, 30000},
                {autoredirect, true}],
    case httpc:request(get, Request, HttpOpts, [{body_format, binary}]) of
        {ok, {{_, 200, _}, _, Body}} -> {ok, Body};
        {ok, {{_, Status, Reason}, _, _}} ->
            {error, iolist_to_binary(io_lib:format("HTTP ~p ~s for ~s", [Status, Reason, Url]))};
        {error, Reason} ->
            {error, iolist_to_binary(io_lib:format("~p for ~s", [Reason, Url]))}
    end.

sha256_hex(Bin) ->
    binary:encode_hex(crypto:hash(sha256, Bin), lowercase).

%% Extract the server entry point named by the bundle's manifest.json
%% (server.entry_point, e.g. server/stackql) into Dest/<basename>, mode
%% 0755, written to a temp name and renamed into place. Returns the path.
extract_bundle(Bundle, Dest) ->
    try
        {ok, [{_, ManifestBin}]} =
            zip:extract(Bundle, [{file_list, ["manifest.json"]}, memory]),
        #{<<"server">> := #{<<"entry_point">> := Entry}} = json:decode(ManifestBin),
        EntryList = unicode:characters_to_list(Entry),
        case lists:member("..", filename:split(EntryList)) orelse
             filename:pathtype(EntryList) =/= relative of
            true -> throw({bad_entry_point, Entry});
            false -> ok
        end,
        {ok, [{_, BinData}]} = zip:extract(Bundle, [{file_list, [EntryList]}, memory]),
        DestList = unicode:characters_to_list(Dest),
        ok = filelib:ensure_dir(filename:join(DestList, "x")),
        Target = filename:join(DestList, filename:basename(EntryList)),
        Tmp = Target ++ ".tmp-" ++ os:getpid(),
        ok = file:write_file(Tmp, BinData),
        ok = file:change_mode(Tmp, 8#755),
        case file:rename(Tmp, Target) of
            ok -> ok;
            {error, _} ->
                %% Lost a race with a concurrent extractor; the winner's file
                %% is the same bytes.
                _ = file:delete(Tmp),
                true = filelib:is_regular(Target)
        end,
        {ok, unicode:characters_to_binary(Target)}
    catch
        Class:Reason ->
            {error, iolist_to_binary(io_lib:format("~p:~p", [Class, Reason]))}
    end.

%% Run Exe with Args, relaying this process's stdin to the child and the
%% child's stdout back, byte for byte (latin1 device encoding so UTF-8 JSON
%% passes through untouched). The child's stderr is inherited. Returns the
%% child's exit status; when stdin reaches EOF the port is closed (the
%% server exits on transport close) and 0 is returned.
relay(Exe, Args) ->
    ok = io:setopts(standard_io, [binary, {encoding, latin1}]),
    Port = erlang:open_port({spawn_executable, unicode:characters_to_list(Exe)},
                            [{args, [unicode:characters_to_list(A) || A <- Args]},
                             binary, stream, use_stdio, exit_status]),
    Parent = self(),
    _Reader = spawn_link(fun() -> stdin_loop(Port, Parent) end),
    out_loop(Port).

stdin_loop(Port, Parent) ->
    case io:get_line(standard_io, "") of
        eof -> Parent ! stdin_eof;
        {error, _} -> Parent ! stdin_eof;
        Data -> erlang:port_command(Port, Data), stdin_loop(Port, Parent)
    end.

out_loop(Port) ->
    receive
        {Port, {data, Data}} ->
            ok = io:put_chars(standard_io, Data),
            out_loop(Port);
        {Port, {exit_status, Status}} ->
            Status;
        stdin_eof ->
            catch erlang:port_close(Port),
            0
    end.
