

# CICD

Summary:

- At present, PR checks, build and test are all performed through [.github/workflows/build.yml](/.github/workflows/build.yml).
- Releasing over various channels (website, homebrew, chocolatey...) is performed manually.
- Docker Build and Push Jobs have scope for improvement. 
    - These are currently based loosely on patterns described in:
        - https://docs.docker.com/build/ci/github-actions/multi-platform/#distribute-build-across-multiple-runners
        - https://docs.docker.com/build/ci/github-actions/share-image-jobs/ 
    - This pattern does the below:
        - (a) Build and push by digest.
        - (b) Leverage [`docker buildx imagetools`](https://docs.docker.com/reference/cli/docker/buildx/imagetools/) to write desired tags.
    - This pattern is only required because if tag pushes are done concurrently, then identical multi-architecture tags are clobbered in a reverse race condition.
    - **NOTE**.  Be careful selecting from [the available github actions runners](https://github.com/actions/runner-images).  Some runnner instances may be incompatible with assumed toolchain.
- Docker images:
    - [production image at `stackql/stackql`](https://hub.docker.com/r/stackql/stackql/tags).
    - [development (zero guarantees) image at `stackql/stackql-devel`](https://hub.docker.com/r/stackql/stackql-devel/tags).



## Secrets

In lieu of a full implementation of [github actions best practices](https://securitylab.github.com/research/github-actions-preventing-pwn-requests/), currently:
- Pull request checks use [the pull_request event](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request), as per [best practices](https://securitylab.github.com/research/github-actions-preventing-pwn-requests/).
- Integration test steps, which require secrets, leverage [the github context](https://docs.github.com/en/actions/learn-github-actions/contexts#github-context) to avoid running where secrets are absent.
    - Therefore, external fork PRs, which are the community contribution model, do not run integration tests.  Strategically, we may funnel community contribution though a staging branch and/or adopt release branches.  This is not an urgent consideration, and we shall decide after some reflection.

## API mocking

According to [this swagger-codegen example](https://github.com/swagger-api/swagger-codegen/blob/master/bin/python-flask-petstore.sh), it is not overly difficult to generate python mocks from openapi docs.  This can then be used for credible regression testing against new provider docs and certainly the relationship of endpoints to stackql resources.  Know weakness: will not detect defective transform from source (eg: MS-graph, AWS) to openapi.
