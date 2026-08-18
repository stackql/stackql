%% Minimal host-fact FFI for the conformance test: OS family, machine
%% architecture, and home directory. Kept out of the pure library core, which
%% takes these as plain string/function parameters so it stays unit-testable
%% without touching the host.
-module(stackql_mcp_ffi).
-export([os_family/0, arch/0, home/0]).

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
