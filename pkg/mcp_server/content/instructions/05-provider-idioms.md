# Provider idioms

Provider-specific behavior that discovery (`list_methods`, `describe_method`) cannot convey, grouped by provider family. A family gets a section when it earns its first idiom; genuinely provider-independent rules go under "Across providers".

## AWS

AWS has two providers; choose deliberately:

- `aws` wraps the native service APIs (EC2, S3, IAM, and so on). List operations return rich attributes in a single call per region, so prefer `aws` for inventory, audit sweeps and anything that scans many resources. `region` is a required routing param on most resources.
- `awscc` wraps the Cloud Control API: every CloudFormation-supported resource type in a uniform shape. Coverage is broader than `aws` but reads are shallower per call. Resources are paired: `*_list_only` enumerates cheaply (identifiers only, no detail), and the keyed resource requires an `Identifier` in the `WHERE` clause, returning full typed attributes at one request per identifier.
- Rule of thumb: broad attribute scan -> `aws`; resource types `aws` does not expose, point reads by identifier, or enumerate-then-selective-detail -> `awscc` (`*_list_only` for the enumeration, keyed detail only for the identifiers you need).
- `awscc` (Cloud Control) UPDATE uses JSON Patch semantics that `describe_method` cannot convey: the update body field takes an RFC 6902 patch array - operations against the resource's current state, not a full desired state - passed as a JSON string wrapped in `string(...)`. On `awscc`: `UPDATE awscc.<service>.<resource> SET PatchDocument = string('[{"op":"replace","path":"/RetentionInDays","value":180}]') WHERE region = '<region>' AND Identifier = '<id>'`. On the generic surface the same field is `PatchDocument`, keyed by `TypeName` and `Identifier`.
- Cloud Control mutations (`awscc` INSERT/UPDATE/DELETE) are asynchronous: the statement returns a progress event for the despatched request, not the final resource state. Always end the statement with `RETURNING Identifier, Operation, OperationStatus, RequestToken, RetryAfter, StatusMessage`, and never report the change as complete on despatch.
- After despatching an `awscc` mutation, follow it up as a matter of routine: poll `SELECT OperationStatus, StatusMessage, ErrorCode, ResourceModel FROM awscc.cloud_control.resource_request WHERE RequestToken = '<token>' AND region = '<region>'` (waiting roughly `RetryAfter` seconds between polls) until `OperationStatus` is `SUCCESS` or `FAILED`. On `SUCCESS`, confirm by reading `ResourceModel` or re-reading the resource; on `FAILED`, relay `StatusMessage` and `ErrorCode` to the operator. If the token was lost, scan recent requests: `SELECT Identifier, OperationStatus, StatusMessage, ErrorCode FROM awscc.cloud_control.resource_requests WHERE ResourceRequestStatusFilter = '{"OperationStatuses": ["FAILED"], "Operations": ["UPDATE"]}' AND region = '<region>'` (both filter lists accept multiple values). The query library carries parameterized forms of both: `aws/cloud_control/resource-request-by-token` and `aws/cloud_control/resource-requests-by-status`.

## Across providers

- Some APIs only populate certain output fields when specific input params are present on the request. If an expected output column is consistently null, check `describe_method` for optional input params and supply one (eg S3 `bucket_region` is only populated when the list request carries at least one query parameter, such as `"max-buckets"`).
