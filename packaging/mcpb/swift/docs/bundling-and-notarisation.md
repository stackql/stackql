# Bundling and notarising a Mac app with the embedded StackQL binary

This is the document that is half the reason this repo exists. It covers
shipping a signed, notarised macOS `.app` that carries the StackQL MCP
server binary inside it. The thing that makes this possible, and that the
Node/Python MCP server crowd structurally cannot do, is that the published
`stackql` darwin binary is already Developer ID signed and Apple notarised.
A signed app can bundle an already-notarised helper and pass its own
notarisation with that helper intact.

## Why this works (and why npm/pypi wrappers cannot)

Apple notarisation staples a ticket to a `.app` and requires every Mach-O
inside it to be signed with a Developer ID certificate and to have a valid
code signature. A Node or Python MCP server is not a single signed Mach-O;
it is an interpreter plus a tree of scripts and native modules that the app
author did not sign and cannot notarise as a unit. The StackQL server is a
single Go binary that StackQL already signs and notarises at release time.
When you drop it into your `.app`, its own signature and notarised cdhash
travel with it, so your app's notarisation submission succeeds.

## Where to put the binary

Ship the universal binary inside the app bundle. Two conventional
locations, both recognised by `BinaryResolver`:

- `YourApp.app/Contents/Resources/stackql`
- `YourApp.app/Contents/Helpers/stackql`

`Resources/` is the simplest. `Helpers/` reads more clearly as "an
auxiliary executable" and is what some apps prefer. The resolver checks
both. Resources inside a notarised app are not quarantined, so the bundled
path needs no `com.apple.quarantine` handling at runtime - unlike the
download-at-runtime path, which `Quarantine.clear` handles.

In Xcode, add the binary as a "Copy Files" build phase targeting
"Resources" (or an absolute `Contents/Helpers` destination), with "Code
Sign On Copy" left OFF for the StackQL binary - it is already signed, and
re-signing it would invalidate its notarised cdhash. You sign the
surrounding app, not the already-signed helper.

## The Developer ID signing and notarisation flow

The app is distributed outside the App Store (Developer ID), not
sandboxed - see "Sandbox reality check" below. The flow:

1. Build the `.app` with the StackQL binary copied into
   `Contents/Resources/` (or `Contents/Helpers/`).

2. Verify the embedded binary's own signature survived the copy. It must
   still validate independently before you sign the app:

   ```
   codesign --verify --strict --verbose=2 \
     YourApp.app/Contents/Resources/stackql
   # expected: ...stackql: valid on disk
   #           ...stackql: satisfies its Designated Requirement
   ```

   Confirm it is the notarised, Developer ID signed StackQL binary:

   ```
   codesign --display --verbose=4 \
     YourApp.app/Contents/Resources/stackql 2>&1 | grep -E 'Authority|TeamIdentifier|flags'
   # expected: Authority=Developer ID Application: StackQL ...
   #           Authority=Developer ID Certification Authority
   #           Authority=Apple Root CA
   #           TeamIdentifier=<StackQL team id>
   #           flags=0x10000(runtime)   <- hardened runtime
   ```

   `spctl` against the bare binary shows it was notarised by StackQL:

   ```
   spctl --assess --type execute --verbose=4 \
     YourApp.app/Contents/Resources/stackql
   # expected: ...stackql: accepted
   #           source=Notarized Developer ID
   ```

3. Sign the app. Do NOT use `codesign --deep` to re-sign the embedded
   binary. `--deep` recursively re-signs nested code, which would strip the
   StackQL binary's Developer ID signature and break its notarised cdhash.
   Sign inside-out instead, leaving the already-signed binary alone, and
   sign the app wrapper with the hardened runtime:

   ```
   codesign --force --options runtime --timestamp \
     --sign "Developer ID Application: Your Org (TEAMID)" \
     YourApp.app
   ```

   If your app has other unsigned nested code (frameworks you built), sign
   those individually first, then the app wrapper last. The point is
   surgical signing, not `--deep`.

4. Notarise the app (the embedded binary is already notarised; you are
   notarising the wrapper):

   ```
   ditto -c -k --keepParent YourApp.app YourApp.zip
   xcrun notarytool submit YourApp.zip \
     --keychain-profile "AC_NOTARY" --wait
   # expected: status: Accepted
   ```

5. Staple the ticket and do the final end-to-end Gatekeeper check:

   ```
   xcrun stapler staple YourApp.app
   spctl --assess --type execute --verbose=4 YourApp.app
   # expected: YourApp.app: accepted
   #           source=Notarized Developer ID
   ```

After step 5, the embedded binary still validates on its own (re-run the
step 2 commands against the binary inside the stapled app); its signature
and notarised cdhash were never touched.

## Sandbox reality check

The demo app (CloudLens) ships non-sandboxed and Developer ID distributed.
The App Sandbox blocks the two things this app must do:

- Spawn a child process (the StackQL server). The sandbox does not allow a
  sandboxed app to fork/exec an arbitrary helper, and there is no
  entitlement that restores general subprocess spawning.
- Make the outbound network connections StackQL needs to reach provider
  APIs and the StackQL provider registry.

`com.apple.security.network.client` would cover the app's own outbound
calls, but it does not extend to a spawned non-sandboxed child, and the
spawn itself is the harder blocker. Rather than fight the sandbox, v1
ships non-sandboxed with the hardened runtime and Developer ID
distribution, which is a fully supported distribution mode for tools like
this. App Store distribution (which requires the sandbox) is out of scope
for v1.

## Quarantine and the download-at-runtime path

Resources inside a notarised app are not quarantined, so a shipping app
that bundles the binary never sees `com.apple.quarantine`. The fallback
path in `BinaryResolver` that downloads the pin-verified bundle at runtime
does write a file that can carry the attribute; `Quarantine.clear` removes
it after a verified download. Shipping apps should bundle the binary and
set `Options.allowDownload = false` so they never depend on the network or
the quarantine path at runtime.

## Verification transcript checklist

Keep the output of these in your release notes for each shipped build:

- `codesign --verify --strict --verbose=2` on the embedded binary (valid)
- `codesign --display --verbose=4` on the embedded binary (Developer ID +
  hardened runtime, StackQL team id)
- `spctl --assess --type execute` on the embedded binary (Notarized
  Developer ID)
- `notarytool submit --wait` on the app (Accepted)
- `spctl --assess --type execute` on the stapled app (Notarized Developer
  ID)
- the embedded-binary checks again, run against the binary inside the
  final stapled app, to prove signing the wrapper did not disturb it
