#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLUSTER_NAME="operator-integration-test-cluster"
OPERATOR_NAMESPACE="meshery"
OPERATOR_IMAGE="meshery/meshery-operator:integration-test"

check_dependencies() {
  # Check for docker
  if ! command -v docker &> /dev/null; then
    echo "❌ docker is not installed. Please install docker first."
    exit 1
  fi
  echo "✅ docker is installed;"

  # Check for kind
  if ! command -v kind &> /dev/null; then
    echo "❌ kind is not installed. Please install KinD first."
    exit 1
  fi
  echo "✅ kind is installed;"

  # Check for kubectl
  if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed. Please install kubectl first."
    exit 1
  fi
  echo "✅ kubectl is installed;"
}

build_operator_image() {
  echo "🔨 Building operator image..."
  cd "$PROJECT_ROOT"
  
  # Build the operator image (bypassing tests for integration testing)
  DOCKER_BUILDKIT=1 docker build --no-cache -t "$OPERATOR_IMAGE" .
  
  echo "✅ Operator image built: $OPERATOR_IMAGE"
}

assert_resources_meshsync() {
  echo "🔍 Asserting meshsync deployment..."
  
  echo "Waiting for meshsync deployment to be created by operator..."
  timeout=300
  while [ $timeout -gt 0 ]; do
    if kubectl --namespace "$OPERATOR_NAMESPACE" get deployment/meshery-meshsync >/dev/null 2>&1; then
      echo "✅ meshsync deployment found"
      break
    fi
    echo "Waiting for meshsync deployment... ($timeout seconds remaining)"
    sleep 5
    timeout=$((timeout - 5))
  done
  
  if [ $timeout -le 0 ]; then
    echo "❌ meshsync deployment was not created within timeout"
    exit 1
  fi
  
  echo "Now waiting for meshsync deployment to be ready..."
  kubectl --namespace "$OPERATOR_NAMESPACE" wait --for=condition=available --timeout=600s deployment/meshery-meshsync || {
    echo "❌ meshsync deployment failed to become ready"
    kubectl --namespace "$OPERATOR_NAMESPACE" get pods -l app=meshery,component=meshsync
    kubectl --namespace "$OPERATOR_NAMESPACE" describe deployment/meshery-meshsync
    exit 1
  }
  
  echo "✅ Meshsync deployment is ready!"
}

assert_resources_broker() {
  echo "🔍 Asserting broker statefulset..."
  
  echo "Waiting for broker statefulset to be created by operator..."
  timeout=300
  while [ $timeout -gt 0 ]; do
    if kubectl --namespace "$OPERATOR_NAMESPACE" get statefulset/meshery-broker >/dev/null 2>&1; then
      echo "✅ broker statefulset found"
      break
    fi
    echo "Waiting for broker statefulset... ($timeout seconds remaining)"
    sleep 5
    timeout=$((timeout - 5))
  done
  
  if [ $timeout -le 0 ]; then
    echo "❌ broker statefulset was not created within timeout"
    exit 1
  fi
  
  echo "Now waiting for broker statefulset to be ready..."
  kubectl --namespace "$OPERATOR_NAMESPACE" wait --for=jsonpath='{.status.readyReplicas}'=1 --timeout=600s statefulset/meshery-broker || {
    echo "❌ broker statefulset failed to become ready"
    kubectl --namespace "$OPERATOR_NAMESPACE" get pods -l app=meshery,component=broker
    kubectl --namespace "$OPERATOR_NAMESPACE" describe statefulset/meshery-broker
    exit 1
  }
  
  echo "✅ Broker statefulset is ready!"
}

assert_resources_cr_broker_status() {
  echo "🔍 Asserting broker CR status property..."
  
  echo "Waiting for broker CR to have endpoints in status..."
  timeout=300
  while [ $timeout -gt 0 ]; do
    external_endpoint=$(kubectl --namespace "$OPERATOR_NAMESPACE" get broker meshery-broker -o jsonpath='{.status.endpoint.external}' 2>/dev/null)
    internal_endpoint=$(kubectl --namespace "$OPERATOR_NAMESPACE" get broker meshery-broker -o jsonpath='{.status.endpoint.internal}' 2>/dev/null)
    
    if [ -n "$external_endpoint" ] && [ -n "$internal_endpoint" ]; then
      echo "✅ broker CR status has external endpoint: $external_endpoint"
      echo "✅ broker CR status has internal endpoint: $internal_endpoint"
      break
    fi
    echo "Waiting for broker CR endpoints... ($timeout seconds remaining)"
    sleep 5
    timeout=$((timeout - 5))
  done
  
  if [ $timeout -le 0 ]; then
    echo "❌ broker CR endpoints were not populated within timeout"
    kubectl --namespace "$OPERATOR_NAMESPACE" get broker meshery-broker -o yaml
    exit 1
  fi
  
  echo "✅ Broker CR endpoint validation completed!"
}

assert_resources() {
  echo "🔍 Asserting operator functionality..."
  
  assert_resources_meshsync
  assert_resources_broker
  assert_resources_cr_broker_status
  
  echo "✅ All components (operator, meshsync, broker) are deployed and ready!"
  echo "✅ Operator functionality assertion completed successfully!"
}

setup() {
  check_dependencies
  echo "🔧 Setting up..."

  echo "Creating KinD cluster..."
  kind create cluster --name "$CLUSTER_NAME"

  echo "Loading operator image into KinD cluster..."
  build_operator_image
  kind load docker-image "$OPERATOR_IMAGE" --name "$CLUSTER_NAME"

  # Pre-load the workload images the operator deploys so the readiness asserts
  # are not gated on a first-time image pull on the kind node. The meshsync
  # image in particular is large and its pull latency from Docker Hub varies
  # widely on CI runners (observed 2s-5min), which previously blew past the
  # readiness budget. Pulling here (on the runner, with a warm network) and
  # side-loading into kind makes pod startup deterministic.
  echo "Pre-loading workload images into KinD cluster..."
  for img in meshery/meshsync:stable-latest; do
    if docker pull "$img"; then
      kind load docker-image "$img" --name "$CLUSTER_NAME" || echo "⚠️  failed to side-load $img (will pull at runtime)"
    else
      echo "⚠️  failed to pull $img (will pull at runtime)"
    fi
  done

  echo "Creating $OPERATOR_NAMESPACE namespace..."
  kubectl create namespace "$OPERATOR_NAMESPACE" || true

  echo "Installing operator CRDs..."
  cd "$PROJECT_ROOT"
  make install

  echo "Deploying operator to cluster..."
  cd "$PROJECT_ROOT"
  
  # Create temporary config directory
  TEMP_CONFIG_DIR=$(mktemp -d)
  cp -r config/* "$TEMP_CONFIG_DIR/"
  
  # Set the image in temporary config
  echo "Setting operator image to: $OPERATOR_IMAGE"
  cd "$TEMP_CONFIG_DIR/manager" 
  "$PROJECT_ROOT/bin/kustomize" edit set image meshery/meshery-operator="$OPERATOR_IMAGE"
  
  # Set imagePullPolicy to Never for integration tests (image is loaded into kind cluster).
  # Use a portable in-place edit: GNU sed accepts a bare `-i`, but BSD/macOS sed
  # requires an explicit backup suffix, so pass one and delete the backup.
  sed -i.bak 's/imagePullPolicy: Always/imagePullPolicy: Never/' manager.yaml && rm -f manager.yaml.bak
  
  cd "$PROJECT_ROOT"
  
  # Build and deploy using temporary config
  make manifests kustomize
  "$PROJECT_ROOT/bin/kustomize" build "$TEMP_CONFIG_DIR/default" | kubectl apply -f -
  
  # Clean up temporary directory
  rm -rf "$TEMP_CONFIG_DIR"

  echo "Applying test resources using existing samples..."
  kubectl --namespace "$OPERATOR_NAMESPACE" apply -f "$PROJECT_ROOT/config/samples/meshery_v1alpha1_broker.yaml"
  kubectl --namespace "$OPERATOR_NAMESPACE" apply -f "$PROJECT_ROOT/config/samples/meshery_v1alpha1_meshsync.yaml"

  echo "Waiting for operator to be ready..."
  kubectl --namespace "$OPERATOR_NAMESPACE" rollout status deployment/meshery-operator --timeout=300s

  echo "Describing operator pod to verify image..."
  kubectl --namespace "$OPERATOR_NAMESPACE" describe pod -l app=meshery,component=operator
  
  echo "✅ Setup completed - operator deployed and CRs applied"

  echo "Outputting cluster resources..."
  echo "--- Operator namespace resources ---"
  kubectl --namespace "$OPERATOR_NAMESPACE" get all
  echo "--- Custom Resources ---"
  kubectl --namespace "$OPERATOR_NAMESPACE" get brokers,meshsyncs
}

cleanup() {
  echo "🧹 Cleaning up..."

  echo "Deleting KinD cluster..."
  kind delete cluster --name "$CLUSTER_NAME"

  echo "Removing operator image..."
  docker rmi "$OPERATOR_IMAGE" || true
}

debug_output() {
  echo "=== Pods in $OPERATOR_NAMESPACE namespace ==="
  kubectl get pods -n "$OPERATOR_NAMESPACE" || true
  echo "=== Deployment status ==="
  kubectl get deployment meshery-operator -n "$OPERATOR_NAMESPACE" || true
  echo "=== ReplicaSet status ==="
  kubectl get replicaset -n "$OPERATOR_NAMESPACE" || true
  echo "=== Pod describe ==="
  kubectl describe pods -n "$OPERATOR_NAMESPACE" || true
  echo "=== Pod logs ==="
  kubectl logs deployment/meshery-operator -n "$OPERATOR_NAMESPACE" --tail=100 || true
}

print_help() {
  echo "Usage: $0 {check_dependencies|setup|assert|cleanup|debug|help}"
}

# Main dispatcher
case "$1" in
  check_dependencies)
    check_dependencies
    ;;
  setup)
    setup
    ;;
  assert)
    assert_resources
    ;;
  cleanup)
    cleanup
    ;;
  debug)
    debug_output
    ;;
  help)
    print_help
    ;;
  *)
    echo "❌ Unknown command: $1"
    print_help
    exit 1
    ;;
esac