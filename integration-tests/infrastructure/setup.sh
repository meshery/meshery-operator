#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CLUSTER_NAME="operator-integration-test-cluster"
OPERATOR_NAMESPACE="meshery"
TEST_NAMESPACE="meshery-test"
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

setup() {
  check_dependencies
  echo "🔧 Setting up..."

  echo "Creating KinD cluster..."
  kind create cluster --name "$CLUSTER_NAME"

  echo "Loading operator image into KinD cluster..."
  build_operator_image
  kind load docker-image "$OPERATOR_IMAGE" --name "$CLUSTER_NAME"

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
  cd "$PROJECT_ROOT"
  
  # Build and deploy using temporary config
  make manifests kustomize
  "$PROJECT_ROOT/bin/kustomize" build "$TEMP_CONFIG_DIR/default" | kubectl apply -f -
  
  # Clean up temporary directory
  rm -rf "$TEMP_CONFIG_DIR"

  echo "Creating $TEST_NAMESPACE namespace..."
  kubectl create namespace "$TEST_NAMESPACE" || true

  echo "Applying test resources using existing samples..."
  kubectl --namespace "$OPERATOR_NAMESPACE" apply -f "$PROJECT_ROOT/config/samples/meshery_v1alpha1_broker.yaml"
  kubectl --namespace "$OPERATOR_NAMESPACE" apply -f "$PROJECT_ROOT/config/samples/meshery_v1alpha1_meshsync.yaml"

  echo "Waiting for operator to be ready..."
  kubectl --namespace "$OPERATOR_NAMESPACE" rollout status deployment/meshery-operator --timeout=300s

  echo "Describing operator pod to verify image..."
  kubectl --namespace "$OPERATOR_NAMESPACE" describe pod -l app=meshery,component=operator

  echo "Outputting cluster resources..."
  echo "--- Operator namespace resources ---"
  kubectl --namespace "$OPERATOR_NAMESPACE" get all
  echo "--- Test namespace resources ---"
  kubectl --namespace "$TEST_NAMESPACE" get all
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

print_help() {
  echo "Usage: $0 {check_dependencies|setup|cleanup|help}"
}

# Main dispatcher
case "$1" in
  check_dependencies)
    check_dependencies
    ;;
  setup)
    setup
    ;;
  cleanup)
    cleanup
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