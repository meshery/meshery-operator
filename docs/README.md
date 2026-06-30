# Meshery Operator - contributor documentation

Internal, contributor-facing documentation for the Meshery Operator. For
end-user documentation (installing and using Meshery and the operator), see
the [Meshery docs site](https://docs.meshery.io).

| Doc | What it covers |
|-----|----------------|
| [architecture.md](architecture.md) | How the operator is structured: the two CRDs, the two controllers, the resource builders, the typed client, and how Meshery Server drives it all. |
| [development.md](development.md) | Local setup, the `go/v4` project layout, the Makefile targets, tool versions, and how to build/run the operator. |
| [testing.md](testing.md) | The three test tiers (unit, envtest, kind e2e) and how to run each. |
| [errors.md](errors.md) | The error-handling convention (MeshKit structured errors) and how to add a new error. |
| [proposals/operator-modernization-plan.md](proposals/operator-modernization-plan.md) | The phased modernization plan and the eight workstreams it is delivered in. |

## Repository layout (Kubebuilder `go/v4`)

```
cmd/main.go                 manager entrypoint (scheme, flags, manager wiring)
api/v1alpha1/               CRD Go types (Broker, MeshSync) + groupversion + deepcopy
controllers/               Broker and MeshSync reconcilers + error registry
pkg/broker/                NATS broker resource builders, health, endpoint derivation
pkg/meshsync/              MeshSync (cluster sync) resource builders, health
pkg/client/v1alpha1/       hand-rolled typed clientset consumed by Meshery Server
pkg/utils/                 small shared helpers
config/                    Kustomize bases (crd, rbac, manager, webhook, certmanager, ...)
bundle/                    OLM bundle (operator catalog packaging)
integration-tests/         bash + kind end-to-end harness
docs/                      this documentation
```
