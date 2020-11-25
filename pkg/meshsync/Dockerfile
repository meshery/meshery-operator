FROM golang:1.13 as builder

WORKDIR /build
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download
# Copy the go source
COPY pkg/meshsync/meshsync.go meshsync.go
COPY pkg/meshsync/cluster/ cluster/
COPY pkg/meshsync/meshes/ meshes/
COPY pkg/meshsync/proto/ proto/
COPY pkg/meshsync/service/ service/
# Build
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o meshery-meshsync meshsync.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/base
WORKDIR /
ENV GODISTRO="debian"
ENV GOARCH="amd64"
COPY --from=builder /build/meshery-meshsync .
ENTRYPOINT ["/meshery-meshsync"]
