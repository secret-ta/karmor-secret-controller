#!/bin/bash

# Set namespace variables
NAMESPACE1="karmor-secret-controller-test-ns-single"
NAMESPACE2="karmor-secret-controller-test-ns-multiple"

# Function to create a custom namespace for testing and apply manifest files
apply_manifests() {
  local namespace=$1
  local manifest_suffix=$2

  # Create a custom namespace for testing
  kubectl create namespace $namespace

  # Create secrets in the namespace
  kubectl create secret generic karmor-secret-controller-test-secret --from-literal=mysecret="test" -n $namespace
  kubectl create secret generic karmor-secret-controller-test-secret2 --from-literal=mysecret2="test2" -n $namespace

  # Apply manifest files
  kubectl apply -f deployment_${manifest_suffix}.yaml -n $namespace
  kubectl apply -f statefulset_${manifest_suffix}.yaml -n $namespace
  kubectl apply -f daemonset_${manifest_suffix}.yaml -n $namespace

  # Wait for all resources to be ready
  sleep 60
}

# Apply manifests for single secret and multiple secrets
apply_manifests $NAMESPACE1 "single_secret"
apply_manifests $NAMESPACE2 "multiple_secrets"

# Function to test secret access
test_secret_access() {
  local namespace=$1
  local resource_type=$2
  local resource_name=$3
  local secret_name=$4
  local secret_path=$5

  if [[ $resource_type == "deploy" ]]; then
    local pods=$(kubectl get pods -n $namespace -l app=$resource_name -o jsonpath='{.items[*].metadata.name}')
  elif [[ $resource_type == "sts" ]]; then
    local pods=$(kubectl get pods -n $namespace -l app=$resource_name -o jsonpath='{.items[*].metadata.name}')
  elif [[ $resource_type == "ds" ]]; then
    local pods=$(kubectl get pods -n $namespace -l app=$resource_name -o jsonpath='{.items[*].metadata.name}')
  fi

  for pod in $pods; do
    output=$(kubectl exec -n $namespace $pod -- cat $secret_path 2>&1)
    if [[ $output == *"Permission denied"* ]]; then
      echo "$namespace [$resource_name] $secret_name access test PASSED"
    else
      echo "$namespace [$resource_name] $secret_name access test FAILED"
    fi
  done
}

# Test secret access for single secret
test_secret_access $NAMESPACE1 "deploy" "karmor-secret-controller-test-deployment" "Secret1" "/etc/secrets/mysecret"
test_secret_access $NAMESPACE1 "sts" "karmor-secret-controller-test-statefulset" "Secret1" "/etc/secrets/mysecret"
test_secret_access $NAMESPACE1 "ds" "karmor-secret-controller-test-daemonset" "Secret1" "/etc/secrets/mysecret"

# Test secret access for multiple secrets
test_secret_access $NAMESPACE2 "deploy" "karmor-secret-controller-test-deployment" "Secret1" "/etc/secrets/mysecret"
test_secret_access $NAMESPACE2 "deploy" "karmor-secret-controller-test-deployment" "Secret2" "/etc/secrets2/mysecret2"
test_secret_access $NAMESPACE2 "sts" "karmor-secret-controller-test-statefulset" "Secret1" "/etc/secrets/mysecret"
test_secret_access $NAMESPACE2 "sts" "karmor-secret-controller-test-statefulset" "Secret2" "/etc/secrets2/mysecret2"
test_secret_access $NAMESPACE2 "ds" "karmor-secret-controller-test-daemonset" "Secret1" "/etc/secrets/mysecret"
test_secret_access $NAMESPACE2 "ds" "karmor-secret-controller-test-daemonset" "Secret2" "/etc/secrets2/mysecret2"

# Modify resources and test secret access again
kubectl scale deploy karmor-secret-controller-test-deployment --replicas=2 -n $NAMESPACE2
sleep 30
test_secret_access $NAMESPACE2 "deploy" "karmor-secret-controller-test-deployment" "Secret1" "/etc/secrets/mysecret"
test_secret_access $NAMESPACE2 "deploy" "karmor-secret-controller-test-deployment" "Secret2" "/etc/secrets2/mysecret2"

kubectl scale sts karmor-secret-controller-test-statefulset --replicas=2 -n $NAMESPACE2
sleep 30
test_secret_access $NAMESPACE2 "sts" "karmor-secret-controller-test-statefulset" "Secret1" "/etc/secrets/mysecret"
test_secret_access $NAMESPACE2 "sts" "karmor-secret-controller-test-statefulset" "Secret2" "/etc/secrets2/mysecret2"

kubectl delete pod -n $NAMESPACE2 -l app=karmor-secret-controller-test-daemonset
sleep 30
test_secret_access $NAMESPACE2 "ds" "karmor-secret-controller-test-daemonset" "Secret1" "/etc/secrets/mysecret"
test_secret_access $NAMESPACE2 "ds" "karmor-secret-controller-test-daemonset" "Secret2" "/etc/secrets2/mysecret2"

# Cleanup
kubectl delete namespace $NAMESPACE1
kubectl delete namespace $NAMESPACE2
