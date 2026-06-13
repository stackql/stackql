//// Provider auth descriptors.
////
//// StackQL authenticates per provider. The no-credentials fixture used across
//// the family is github / null_auth: it lets `pull github` and a handful of
//// read-only services run with no secrets, which is what the conformance test
//// and the pipewatch demo lean on.

import gleam/json
import gleam/list

/// A single provider's auth configuration. `kind` is the StackQL auth type
/// (e.g. "null_auth", "interactive", "service_account"); `provider` is the
/// provider name (e.g. "github", "aws", "google").
pub type Auth {
  Auth(provider: String, kind: String)
}

/// Convenience constructor matching the family idiom:
///   auth.for("github", "null_auth")
pub fn for(provider provider: String, kind kind: String) -> Auth {
  Auth(provider: provider, kind: kind)
}

/// The no-credentials github fixture, used by the conformance test.
pub fn github_null_auth() -> Auth {
  Auth(provider: "github", kind: "null_auth")
}

/// Render a set of auth descriptors as the JSON StackQL expects under its
/// auth config. Returns a json value the caller can embed where needed.
pub fn to_json(auths: List(Auth)) -> json.Json {
  json.object(
    list.map(auths, fn(a) {
      #(a.provider, json.object([#("type", json.string(a.kind))]))
    }),
  )
}
