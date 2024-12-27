
# `stackql` walkthroughs

All markdown documents hereunder, execept those named `README.md`, 
are **provably working** examples of `stackql` in action.
These materials serve as useful examples and reference materials for
using `stackql`.  If you have some use case that you would like to see added here; please let us know! 


All walkthrough files are testable with CI, leveraging annotations (eg: code block info strings)
in order to setup, run, verify and tear down testing scenarios.  The tests *can* be run:

- Locally and manually, on your own machine.  That's the whole idea; please follow the instructions, mix and match, and let us know any ideas that occur.
- Directly from CI.  Reports are generated and archived.
- From test harnesses, such as robot framework.  This has not yet been implemented.

## Running from CI

The canonical, **ruleset-protected** tag form is `scenario-<run_number>-<anything>`.  At this stage, `run_number` must refer to a `stackql` run for which a `linux` `amd64` stackql binary archive is present at the time the tag is run.  



