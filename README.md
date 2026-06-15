<!-- web assets -->
[logo]: https://stackql.io/img/stackql-logo-bold.png "stackql logo"
[homepage]: https://stackql.io/
[docs]: https://stackql.io/docs
[blog]: https://stackql.io/blog
[registry]: https://github.com/stackql/stackql-provider-registry
[variables]: https://stackql.io/docs/getting-started/variables
[macpkg]: https://storage.googleapis.com/stackql-public-releases/latest/stackql_darwin_multiarch.pkg
[winmsi]: https://releases.stackql.io/stackql/latest/stackql_windows_amd64.msi
[winzip]: https://releases.stackql.io/stackql/latest/stackql_windows_amd64.zip
[tuxzip]: https://releases.stackql.io/stackql/latest/stackql_linux_amd64.zip
<!-- docker links -->
[dockerhub]: https://hub.docker.com/u/stackql
[dockerstackql]: https://hub.docker.com/r/stackql/stackql
[dockerjupyter]: https://hub.docker.com/r/stackql/stackql-jupyter-demo
<!-- github actions links -->
[setupaction]: https://github.com/marketplace/actions/stackql-studios-setup-stackql
[execaction]: https://github.com/marketplace/actions/stackql-studios-stackql-exec
<!-- badges -->
[platforms]: https://img.shields.io/badge/platform-windows%20macos%20linux-brightgreen?style=flat-square "Platforms"
[license]: https://img.shields.io/badge/license-MIT-blue?style=flat-square "License"
[build]: https://github.com/stackql/stackql/actions/workflows/build.yml/badge.svg "Build"
[stars]: https://img.shields.io/github/stars/stackql/stackql?style=flat-square "GitHub Stars"
[forks]: https://img.shields.io/github/forks/stackql/stackql?style=flat-square "GitHub Forks"
[contributors]: https://img.shields.io/github/contributors/stackql/stackql?style=flat-square "Contributors"
[mcpregistrybadge]: https://img.shields.io/badge/MCP%20Registry-io.github.stackql%2Fstackql--mcp-blue?style=flat-square "MCP Registry"
<!-- github links -->
[issues]: https://github.com/stackql/stackql/issues/new?assignees=&labels=bug&template=bug_report.md&title=%5BBUG%5D
[features]: https://github.com/stackql/stackql/issues/new?assignees=&labels=enhancement&template=feature_request.md&title=%5BFEATURE%5D
[developers]: /docs/developer_guide.md
[registrycont]: /docs/registry_contribution.md
[designdocs]: /docs/high-level-design.md
[contributing]: /CONTRIBUTING.md
[discussions]: https://github.com/orgs/stackql/discussions
<!-- repo assets -->
[darkmodeterm]: /docs/images/stackql-light-term.gif#gh-dark-mode-only
[lightmodeterm]: /docs/images/stackql-dark-term.gif#gh-light-mode-only
<!-- misc links -->
[twitter]: https://twitter.com/stackql
[mcpregistry]: https://registry.modelcontextprotocol.io/?search=stackql
[mcpdocs]: https://stackql.io/docs

<!-- language: lang-none -->
<div align="center">

[![logo]][homepage]  
![platforms]
![license]
![build]
[![MCP Registry][mcpregistrybadge]][mcpregistry]
![stars]
![forks]
![contributors]

</div>

<div align="center">

<!-- ![homebrew downloads](https://img.shields.io/homebrew/installs/dy/stackql?label=homebrew%20downloads) -->
![GitHub Release](https://img.shields.io/github/v/release/stackql/stackql?label=github%20release)
![GitHub Downloads](https://img.shields.io/github/downloads/stackql/stackql/total?label=github%20downloads)
![Docker Image Version](https://img.shields.io/docker/v/stackql/stackql?label=docker%20version)
![Docker Pulls](https://img.shields.io/docker/pulls/stackql/stackql)
![Chocolatey Downloads](https://img.shields.io/chocolatey/dt/stackql?label=chocolatey%20downloads)
![Chocolatey Version](https://img.shields.io/chocolatey/v/stackql)
![homebrew version](https://img.shields.io/homebrew/v/stackql?label=homebrew%20version)
<!-- ![PyPI - Downloads](https://img.shields.io/pypi/dm/stackql-deploy?label=pypi%20downloads) -->

</div>
<div align="center">

### Deploy, manage and query cloud resources and interact with APIs using SQL
<!-- <h3 align="center">SQL based XOps, observability and middleware framework</h3> -->

<p align="center">

[__Read the docs »__][docs]  
[Raise an Issue][issues] · 
[Request a Feature][features] · 
[Developer Guide][developers] · 
[BYO Providers][registrycont]

</p>
</div>

<details open="open">
<summary>Contents</summary>
<ol>
<li><a href="#about-the-project">About The Project</a></li>
<li><a href="#mcp-server">MCP Server</a></li>
<li><a href="#installation">Installation</a></li>
<li><a href="#usage">Usage</a></li>
<!-- <li><a href="#roadmap">Roadmap</a></li> -->
<li><a href="#contributing">Contributing</a></li>
<li><a href="#license">License</a></li>
<li><a href="#contact">Contact</a></li>
<li><a href="#acknowledgements">Acknowledgements</a></li>
</ol>
</details>

## About The Project

[__StackQL__][homepage] is an open-source project built with Golang that allows you to create, modify and query the state of services and resources across different local and remote interfaces, using SQL semantics.  Such interfaces canonically include, but are not limited to, cloud and SaaS providers (Google, AWS, Azure, Okta, GitHub, etc.).
<br />
<br />

![stackql-shell][darkmodeterm]
![stackql-shell][lightmodeterm]

### How it works

StackQL is a standalone application that can be used in client mode (via __`exec`__ or __`shell`__) or accessed via a Postgres wire protocol client (`psycopg2`, etc.) using server mode (__`srv`__).  

StackQL parses SQL statements and transpiles them into API requests to the (cloud) resource provider.  The API calls are then executed and the results are returned to the user.  

StackQL provider interfaces are canonically defined in OpenAPI extensions to the providers' specification.  These definitions are then used to generate the SQL schema and the API client.  The source for the provider definitions are stored in the [__StackQL Registry__][registry].  The semantics of provider interactions are defined in [our `any-sdk` library](https://github.com/stackql/any-sdk).  For more detail on nuts and bolts, please see [the local `AGENTS.md` file](/AGENTS.md) and [that of `any-sdk`](https://github.com/stackql/any-sdk/blob/main/AGENTS.md).

<details>
<summary><b>StackQL Context Diagram</b></summary>
<br />
The following context diagram describes the StackQL architecture at a high level:  

<!-- ![StackQL Context Diagram](http://www.plantuml.com/plantuml/proxy?cache=no&src=https://raw.githubusercontent.com/stackql/test-readme/main/puml/stackql-c4-context.iuml) -->

```mermaid
flowchart LR
  subgraph StackQL
    direction BT
    subgraph ProviderDefs
        Registry[Provider Registry Docs]    
    end
    subgraph App
        Proc[$ stackql exec\n$ stackql shell\n$ stackql srv]
        style Proc fill:#000,stroke:#000,color:#fff,text-align:left;

        %% ,font-family:'Courier New', Courier, monospace
    end
  end
  User((User)) <--> StackQL <--> Provider[Cloud Provider API]
  ProviderDefs --> App
```

More detailed design documentation can be found in the [here][designdocs].

</details>

## MCP Server

StackQL is an [MCP](https://modelcontextprotocol.io) server - SQL over 40+ cloud and SaaS providers for AI agents.  Point any MCP-capable client (Claude, VS Code, Cursor, etc.) at StackQL and it can query and provision cloud resources using SQL.

Run it over stdio:

```sh
stackql mcp --mcp.server.type=stdio
```

StackQL MCP is published to the [Official MCP Registry][mcpregistry] as `io.github.stackql/stackql-mcp` and distributed via npm, PyPI, Docker, a GitHub Action and `.mcpb` bundles.  Pick an install vector and add the matching block to your MCP client config:

<details>
<summary><b>npm (npx)</b></summary>

```json
{
  "mcpServers": {
    "stackql": {
      "command": "npx",
      "args": ["-y", "@stackql/mcp-server"]
    }
  }
}
```

</details>

<details>
<summary><b>PyPI (uvx)</b></summary>

```json
{
  "mcpServers": {
    "stackql": {
      "command": "uvx",
      "args": ["stackql-mcp-server"]
    }
  }
}
```

</details>

<details>
<summary><b>Docker</b></summary>

```json
{
  "mcpServers": {
    "stackql": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "stackql/stackql-mcp"]
    }
  }
}
```

</details>

<details>
<summary><b>.mcpb bundle</b></summary>

> Download the bundle for your platform from the [latest release](https://github.com/stackql/stackql/releases/latest) and install it in your MCP client (one-click in clients that support `.mcpb`).

- `stackql-mcp-linux-x64.mcpb`
- `stackql-mcp-linux-arm64.mcpb`
- `stackql-mcp-windows-x64.mcpb`
- `stackql-mcp-darwin-universal.mcpb`

</details>

<details>
<summary><b>GitHub Actions</b></summary>

> Wire StackQL MCP into agentic CI workflows.  Defaults to `read_only` mode, the safe default for CI.

```yaml
- uses: stackql/setup-stackql-mcp@v1
  with:
    mode: read_only
```

</details>

For client-specific setup, authentication and server modes, see the [full MCP install and usage docs][mcpdocs].

## Installation

StackQL is available for Windows, MacOS, Linux, Docker, GitHub Actions and more.  See the installation instructions below for your platform.  

<details>
<summary><b>Installing on MacOS</b></summary>

- Homebrew (`amd64` and `arm64`)
  - `brew install stackql` *or* `brew tap stackql/tap && brew install stackql/tap/stackql`
- MacOS PKG Installer (`amd64` and `arm64`)
  - download the latest [MacOS PKG installer for StackQL][macpkg]
  - run the installer and follow the prompts

</details>

<details>
<summary><b>Installing on Windows</b></summary>

- MSI Installer
  - download the latest [MSI installer for StackQL][winmsi]
  - run the installer and follow the prompts
- Chocolatey
  - install [Chocolatey](https://chocolatey.org/install)
  - run `choco install stackql`
- ZIP Archive
  - download the latest [Windows ZIP archive for StackQL][winzip]
  - extract the archive (code signed `stackql.exe` file) to a directory of your choice
  - add the directory to your `PATH` environment variable (optional)

</details>

<details>
<summary><b>Installing on Linux</b></summary>

- ZIP Archive
  - download the latest [Linux ZIP archive for StackQL][tuxzip]
    - or via `curl -L https://bit.ly/stackql-zip -O && unzip stackql-zip`
  - extract the archive (`stackql` file) to a directory of your choice
  - add the directory to your `PATH` environment variable (optional)

</details>

<details>
<summary><b>Getting StackQL from DockerHub</b></summary>

> View all available StackQL images on [__DockerHub__][dockerhub].  Images available include [__`stackql`__][dockerstackql], [__`stackql-jupyter-demo`__][dockerjupyter] and more.  Pull the latest StackQL base image using:  

```bash
docker pull stackql/stackql
```

</details>

<details>
<summary><b>Using StackQL with GitHub Actions</b></summary>

> Use StackQL in your GitHub Actions workflows to automate cloud infrastructure provisioning, IaC assurance, or compliance/security.  Available GitHub Actions include: [`setup-stackql`][setupaction], [`stackql-exec`][execaction] and more

</details>

## Usage

StackQL can be used via the interactive REPL shell, or via the `exec` command or ran as a server using the [Postgres wire protocol](https://www.postgresql.org/docs/current/protocol.html).  

> ℹ️ StackQL does not require or install a database.

* Interactive Shell
  ```sh
  # run interactive stackql queries
  stackql shell --auth="${AUTH}"
  ```
* Execute a statement or file
  ```sh
  stackql exec --auth="${AUTH}" -i myscript.iql --iqldata vars.jsonnet --output json
  
  # or
  
  stackql exec --auth="${AUTH}" "SELECT id, status FROM aws.ec2.instances WHERE region = 'us-east-1'"
  ```

  > ℹ️ output options of `json`, `csv`, `table` and `text` are available for the `exec` command using the `--output` flag

  > ℹ️ StackQL supports passing parameters using `jsonnet` or `json`, see [__Using Variables__][variables]
* Server
  ```sh
  # serve client requests over the Postgres wire protocol (psycopg2, etc.) 
  stackql srv --auth="${AUTH}"
  ```

_For more examples, please check our [Blog][blog]_

<!-- ## Roadmap

See our [__roadmap__](https://github.com/othneildrew/Best-README-Template/issues) to see where we are going with the project. -->

## Contributing

Contributions are welcome and encouraged.  For more information on how to contribute, please see our [__contributing guide__][contributing].

## License

Distributed under the MIT License. See [`LICENSE`](https://github.com/stackql/stackql/blob/main/LICENSE) for more information.  Licenses for third party software we are using are included in the [/docs/licenses](/docs/licenses) directory.

## Contact

Get in touch with us via Twitter at [__@stackql__][twitter], email us at [__info@stackql.io__](info@stackql.io) or start a conversation using [__discussions__][discussions].

## Acknowledgements
Forks of the following support our work:

* [vitess](https://vitess.io/)
* [kin-openapi](https://github.com/getkin/kin-openapi)
* [gorilla/mux](https://github.com/gorilla/mux)
* [readline](https://github.com/chzyer/readline)
* [psql-wire](https://github.com/jeroenrinzema/psql-wire)
* [mcp-postgres](https://github.com/gldc/mcp-postgres)
* [the `golang` MCP SDK](https://github.com/modelcontextprotocol/go-sdk)
* ...and more.  Please excuse us for any omissions.

We gratefully acknowledge these pieces of work.

