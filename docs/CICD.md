

# CICD

Summary:

- At present, PR checks, build and test are all performed through [.github/workflows/go.yml](/.github/workflows/go.yml).
- Releasing over various channels (website, homebrew, chocolatey...) is performed manually.
- The strategic state is to split the functions: PR checks, build and test; into separate files, and migrate to use [goreleaser](https://goreleaser.com/).
- Should take the hint from docker to [speed up multi-platform builds using multiple runners](https://docs.docker.com/build/ci/github-actions/multi-platform/#distribute-build-across-multiple-runners).


## Secrets

In lieu of a full implementation of [github actions best practices](https://securitylab.github.com/research/github-actions-preventing-pwn-requests/), currently:
- Pull request checks use [the pull_request event](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request), as per [best practices](https://securitylab.github.com/research/github-actions-preventing-pwn-requests/).
- Integration test steps, which require secrets, leverage [the github context](https://docs.github.com/en/actions/learn-github-actions/contexts#github-context) to avoid running where secrets are absent.
    - Therefore, external fork PRs, which are the community contribution model, do not run integration tests.  Strategically, we may funnel community contribution though a staging branch and/or adopt release branches.  This is not an urgent consideration, and we shall decide after some reflection.

## API mocking

According to [this swagger-codegen example](https://github.com/swagger-api/swagger-codegen/blob/master/bin/python-flask-petstore.sh), it is not overly difficult to generate python mocks from openapi docs.  This can then be used for credible regression testing against new provider docs and certainly the relationship of endpoints to stackql resources.  Know weakness: will not detect defective transform from source (eg: MS-graph, AWS) to openapi.
