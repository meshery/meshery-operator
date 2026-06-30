# Contributing to Meshery 

Please do! Thank you for your help in improving Meshery! :balloon:

 Find the complete set of contributor guides at https://docs.meshery.io/project/contributing


## Running Tests Locally

```bash
# Install envtest binaries and run tests
make test

# Run tests with coverage
make coverage
```

If you see `etcd not found in PATH`, ensure you run `make test` (which handles envtest setup automatically) rather than `go test ./...` directly.
