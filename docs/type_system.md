
# __`stackql`__ type system

There exists a relation from openapi document (discovery document) type and relational type, we dub it "discovery-relational mapping (DRM)":

$\ R_{drm}: \text{discovery-type} \to \text{relational-type}\ \ \ \ \ (1) $

In addition, the "traditional" object-relational mapping relation exists:

$\ Q_{orm}: \text{relational-type} \to \text{golang-type}\ \ \ \ \ (2) $

These relations are mapped out in:

- [internal/stackql/drm](/internal/stackql/drm).
- [internal/stackql/typing](/internal/stackql/typing).

The `golang` `sql` driver is used: 

```go
import (
    "database/sql"
)
    

var (
    _ *sql.ColumnType = (*sql.ColumnType)(nil) // This is the golang SQL driver type

)
```