# Security Policy

The StackQL team takes the security of `stackql` seriously. We appreciate everyone who reports vulnerabilities responsibly, and we will do our best to acknowledge and address valid reports promptly.

## Reporting a Vulnerability

Please do not open a public GitHub issue for security reports. A public issue discloses the problem before a fix is available and puts users at risk.

Instead, report privately using GitHub private vulnerability reporting:

1. Open the [new advisory form](https://github.com/stackql/stackql/security/advisories/new) (Security tab -> Advisories -> "Report a vulnerability").
2. Provide as much detail as you can: affected version(s), reproduction steps, impact, and any suggested remediation.
3. Submit. This opens a private advisory thread visible only to you and the maintainers.

If you cannot use GitHub private vulnerability reporting, email us at [info@stackql.io](mailto:info@stackql.io). Please put "SECURITY" in the subject line and treat the contents as confidential.

## Supported Versions

Security fixes are applied to the latest released minor line. Older lines are supported on a best-effort basis only; we recommend upgrading to the latest release. See the [releases page](https://github.com/stackql/stackql/releases) for the current version.

| Version | Supported    |
| ------- | ------------ |
| 0.10.x  | Yes          |
| < 0.10  | Best-effort  |

## Response Expectations

These are targets, not contractual guarantees:

- We aim to acknowledge a report within a few business days.
- We will keep you updated as we investigate and work on a fix.
- We will coordinate disclosure timing with you once a fix or mitigation is ready.

## Disclosure Policy

We follow coordinated (responsible) disclosure. Please give us a reasonable opportunity to release a fix before any public disclosure. We are happy to credit reporters in the advisory and release notes, unless you prefer to remain anonymous.

## Scope

This policy covers the `stackql` engine in this repository and its official distributions and images (for example, the binaries on the releases page and the `stackql/stackql` Docker images). Provider definitions are maintained separately in the [stackql-provider-registry](https://github.com/stackql/stackql-provider-registry), and the request-execution library in [any-sdk](https://github.com/stackql/any-sdk); please report issues specific to those in their respective repositories.
