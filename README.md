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
    use google; SELECT * FROM compute.instance WHERE zone = 'australia-southeast1-b' AND project = 'my-project' ;

----

## Provider development

Keen to expose some new functionality though `stackql`?  We are very keen on this!  

Please see [registry_contribution.md](/docs/registry_contribution.md).

---

## Design

[HLDD](/docs/high-level-design.md)


---

## Providers

- Google.
- Okta.

---

## Build

With cmake:

```bash
cd build
cmake ..
cmake --build .
```


Executable `build/stackql` will be created.


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

## Acknowledgements

Forks of the following support our work:

  - [vitess](https://vitess.io/)
  - [readline](https://github.com/chzyer/readline)
  - [color](https://github.com/fatih/color)

We gratefully acknowledge these pieces of work.

## License

See [/LICENSE](/LICENSE)
