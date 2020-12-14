#!/bin/sh

# To use envtest is required to have etcd, kube-apiserver and kubetcl binaries installed locally.
# This script will create the setup ci required deps for testenv


os=$(go env GOOS)
arch=$(go env GOARCH)
TESTBIN_DIR=testbin

# download kubebuilder and extract it to tmp
curl -L https://go.kubebuilder.io/dl/2.3.1/${os}/${arch} | tar -xz -C /tmp/
mkdir $TESTBIN_DIR
sudo mv /tmp/kubebuilder_2.3.1_${os}_${arch} /usr/local/kubebuilder
export PATH=$TESTBIN_DIR/kubebuilder/bin