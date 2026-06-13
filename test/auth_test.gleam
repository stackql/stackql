//// Unit tests for provider auth descriptors.

import gleam/json
import gleeunit/should
import stackql_mcp/auth

pub fn for_constructs_descriptor_test() {
  auth.for("github", "null_auth")
  |> should.equal(auth.Auth(provider: "github", kind: "null_auth"))
}

pub fn github_null_auth_fixture_test() {
  auth.github_null_auth()
  |> should.equal(auth.Auth(provider: "github", kind: "null_auth"))
}

pub fn to_json_renders_provider_keyed_test() {
  let rendered =
    auth.to_json([auth.for("github", "null_auth")])
    |> json.to_string

  rendered
  |> should.equal("{\"github\":{\"type\":\"null_auth\"}}")
}
