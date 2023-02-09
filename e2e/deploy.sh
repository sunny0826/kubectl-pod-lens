#!/bin/sh

# Deploy the e2e application
kubectl apply -f e2e/deployment.yaml

# rollout status
kubectl rollout status deployment/my-nginx-deployment

# Set the name of the deployment
deployment_name="my-nginx-deployment"

# Get the name of the first pod in the deployment
pod_name=$(kubectl get pods --selector=app=$deployment_name --output=jsonpath='{.items[0].metadata.name}')

echo "The first pod in the deployment '$deployment_name' is named '$pod_name'"

# Test
#bin/pod-lens $pod_name