# Error handling

The operator is standardizing on **MeshKit structured errors**
(`github.com/meshery/meshkit/errors`) as its error-handling convention. The goal is
that every error returned from a controller or package is constructed with a stable
code and rich, user-facing metadata so that Meshery Server and the meshkit
error-reference tooling can surface actionable guidance instead of an opaque string.
New errors must use this form; the existing registry is mid-migration (see
[Migration status](#migration-status) below).

## The convention

Define one exported code constant per error, matching the regex `^Err[A-Z].+Code$`,
and one constructor that wraps the underlying cause:

```go
import "github.com/meshery/meshkit/errors"

// Codes are unique within the component. Allocate the next free code from
// helpers/component_info.json and bump its next_error_code.
const ErrReconcileBrokerCode = "meshery-operator-1006"

func ErrReconcileBroker(err error) error {
    return errors.New(
        ErrReconcileBrokerCode,
        errors.Alert,                                              // Severity
        []string{"Failed to reconcile the Broker custom resource"}, // ShortDescription
        []string{err.Error()},                                     // LongDescription
        []string{"The NATS StatefulSet, Service, or ConfigMaps could not be created or updated"}, // ProbableCause
        []string{"Check the operator RBAC and the events on the owned objects in the broker's namespace"}, // SuggestedRemediation
    )
}
```

`errors.NewDefault(code, ldescription...)` exists for the rare case where only a
code and message are available, but prefer the full `errors.New` form.

## Rules

- **Include the offending resource's name and namespace** in the
  ShortDescription/LongDescription. A reconcile error must say *which* `Broker` or
  `MeshSync` failed - e.g. `MeshSync resource "<name>" configuration invalid` -
  not just `configuration invalid`.
- **Surface errors to status, not just logs.** When reconciliation fails, set a
  `Condition` (and where relevant the `PublishingTo`/`Endpoint` status) carrying
  the structured error, so Meshery Server can observe it - do not only
  `log.Error`/print to stdout.
- **Allocate codes from `helpers/component_info.json`.** Bump `next_error_code`
  when you add one; do not reuse or guess codes.
- **Wrap, don't swallow.** Always pass the underlying `err` into the
  LongDescription (or wrap with `%w` upstream) so the root cause is preserved.

## Migration status

The error registry in `controllers/error.go` historically used `fmt.Errorf` with
bare numeric string codes (`1001`-`1012`, plus an outlier `11049`), while
`pkg/broker/error.go` and `pkg/meshsync/error.go` use the standard-library `errors`
package with concatenated codes (`1013`-`1016`) that currently collide across the
two packages. These are being migrated to the MeshKit form above and renumbered into
the `meshery-operator-NNNN` namespace (allocated from `helpers/component_info.json`)
as the controllers are reworked (WS-3, #785); new errors must use the MeshKit form
from the outset.
