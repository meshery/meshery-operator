#!/bin/sh

# To use envtest is required to have etcd, kube-apiserver and kubetcl binaries installed locally.
# This script will create the setup ci required deps for testenv


os=$(go env GOOS)
arch=$(go env GOARCH)

# download kubebuilder and install locally.
curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env G      OARCH)
chmod +x kubebuilder && mv kubebuilder /usr/local/bin/
