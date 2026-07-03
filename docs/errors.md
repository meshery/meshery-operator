# Error handling

The operator uses **MeshKit structured errors**
(`github.com/meshery/meshkit/errors`) as its error-handling convention - never
`fmt.Errorf` or `errors.New`. Every error
returned from a controller or package is constructed with a stable code and rich,
user-facing metadata so that Meshery Server and the meshkit error-reference tooling
can surface actionable guidance instead of an opaque string.

## The convention

Define one exported code constant per error, matching the regex `^Err[A-Z].+Code$`,
and one constructor that carries the underlying cause. Error names and codes are
unique within the whole component (not just within a Go package), so name each
constructor distinctly - e.g. `ErrGettingBrokerResource` and
`ErrGettingMeshsyncResource`, not two `ErrGettingResource`:

```go
import meshkiterrors "github.com/meshery/meshkit/errors"

// Codes are bare integers, unique within the component. Allocate the next free
// code from helpers/component_info.json and bump its next_error_code.
const ErrReconcileBrokerCode = "1006"

func ErrReconcileBroker(err error) error {
    return meshkiterrors.New(
        ErrReconcileBrokerCode,
        meshkiterrors.Alert,                                       // Severity
        []string{"Broker reconciliation failed"},                  // ShortDescription
        []string{err.Error()},                                     // LongDescription (cause)
        []string{"The NATS StatefulSet, Service, or ConfigMaps could not be created or updated"}, // ProbableCause
        []string{"Check the operator RBAC and the events on the owned objects in the broker's namespace"}, // SuggestedRemediation
    )
}
```

The ShortDescription, ProbableCause, and SuggestedRemediation must be string
literals so the errorutil tool can extract them for the error reference; the dynamic
cause (`err.Error()`) goes in the LongDescription and is not extracted.
`meshkiterrors.NewDefault(code, ldescription...)` exists for the rare case where only
a code and message are available, but it is deprecated - prefer the full
`meshkiterrors.New` form.

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
- **Carry, don't swallow.** Always pass the underlying `err` into the
  LongDescription so the root cause is preserved in the message. MeshKit errors are
  the terminal error type (they do not implement `Unwrap`), so capture the cause at
  construction time.

## Tooling

- `make error` runs the meshkit errorutil analyzer (read-only). It validates that
  codes and names are unique, flags deprecated `NewDefault` usage, and regenerates
  the `errorutil_*.json` reference files under `helpers/` (gitignored). Run it after
  adding or changing an error.
- `make error-util` assigns codes to any new placeholder constants and bumps
  `next_error_code` in `helpers/component_info.json`.

## Status

All three error registries use the MeshKit form: `controllers/error.go` (codes
`1001`-`1012` and `1017`), `pkg/broker/error.go` (`1013`-`1016`), and
`pkg/meshsync/error.go` (`1018`-`1021`). The meshsync registry was renumbered and
renamed from its original `1013`-`1016` to resolve a code-and-name collision with the
broker registry, and the controllers' `ErrMarshal` outlier (`11049`) was renumbered
to `1017`. `make error` reports no duplicate codes or names.
