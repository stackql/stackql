<!-- language: lang-none -->

![Platforms](https://img.shields.io/badge/platform-windows%20macos%20linux-brightgreen)
![Go](https://github.com/stackql/stackql/workflows/Go/badge.svg)
![License](https://img.shields.io/github/license/stackql/stackql)
![Lines](https://img.shields.io/tokei/lines/github/stackql/stackql)  
[![StackQL](https://docs.stackql.io/img/stackql-banner.png)](https://stackql.io/)  


# Deploy, Manage and Query Cloud Infrastructure using SQL

[[Documentation](https://docs.stackql.io/)]  [[Developer Guide](/docs/developer_guide.md)] [[BYO Providers](/docs/registry_contribution.md)]

## Cloud infrastructure coding using SQL

> StackQL allows you to create, modify and query the state of services and resources across all three major public cloud providers (Google, AWS and Azure) using a common, widely known DSL...SQL.

----
## Its as easy as...
    SELECT * FROM google.compute.instance WHERE zone = 'australia-southeast1-b' AND project = 'my-project' ;

----

## Provider development

Keen to expose some new functionality though `stackql`?  We are very keen on this!  

Please see [registry_contribution.md](/docs/registry_contribution.md).

---

## Design

[HLDD](/docs/high-level-design.md)


---

## Providers

Please see [the stackql-provider-registry repository](https://github.com/stackql/stackql-provider-registry)

Providers include:

- Google.
- Okta.
- ...

---

## Build

Presuming you have all of [the system requirements](#system-requirements-for-local-devlopment-build-and-test), then build/test with cmake:

```bash
cd build
cmake ..
cmake --build .
```

Executable `build/stackql` will be created.

### System requirements for local development, build and test

- cmake>=3.22.3
- golang>=1.16
- openssl>=1.1.1
- python>=3.5


## Run

```bash
./build/stackql --help

```

## Examples

```
./stackql exec "show extended services from google where title = 'Service Directory API';"
```

More examples in [docs/examples.md](/docs/examples.md).

---

## Developers

[docs/developer_guide.md](/docs/developer_guide.md).

## Testing

[test/README.md](/test/README.md).

[docs/integration_testing.md](/docs/integration_testing.md).

## Server mode

Please see [the server mode section of the developer docs](/docs/developer_guide.md#server-mode).

## Acknowledgements

Forks of the following support our work:

  - [vitess](https://vitess.io/)
  - [readline](https://github.com/chzyer/readline)
  - [color](https://github.com/fatih/color)

We gratefully acknowledge these pieces of work.

## Licensing

Please see the [stackql LICENSE](/LICENSE).

Licenses for third party software we are using are included in the [/licenses](/licenses) directory.
