
# Testing approach

## Contrived local integration testing

Offline invocations of `stackql` are assessed against expected responses, through:

1. the functionality of [/test/python/main.py](/test/python/main.py).
2. [robot tests in /test/functional](/test/functional)  

(1) is deprecated and will be entirely migrated to (2).

These tests are run during the build process:
  - locally through cmake as per [/README.md#build](/README.md#build)
  - in github actions based CICD as per [/.github/workflows/go.yml](/.github/workflows/go.yml).

## Unit tests using standard golang approaches

Proliferation is a fair way behind development.

These are also run inside build processes: local and remote.

## E2E integration tests

TBA.


## Sundry opinions about testing in golang

  - [Simple approach and dot import.](https://medium.com/@benbjohnson/structuring-tests-in-go-46ddee7a25c)
  - [Making use of containers, make and docker-compose for integration testing.](https://blog.gojekengineering.com/golang-integration-testing-made-easy-a834e754fa4c)
  - [HTTP client testing.](http://hassansin.github.io/Unit-Testing-http-client-in-Go)
  - [Mocking HTTPS in unit tests.](https://stackoverflow.com/questions/27880930/mocking-https-responses-in-go)