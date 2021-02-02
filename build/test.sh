#!/bin/bash

########## Prepare directories for Kubebuilder ##########
sudo mkdir -p /usr/local/kubebuilder
sudo chmod 777 /usr/local/kubebuilder

########## Fetch Kubebuilder ##########
if [ ! -d "/usr/local/kubebuilder/bin" ]; then
  curl -L https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.3.1/kubebuilder_2.3.1_linux_amd64.tar.gz -o kubebuilder.tar.gz
  tar -zxvf kubebuilder.tar.gz --strip-components=1 -C /usr/local/kubebuilder
fi

########## Install kubectl ##########

curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
curl -LO "https://dl.k8s.io/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl.sha256"
echo "$(<kubectl.sha256) kubectl" | sha256sum --check
mkdir -p ~/.local/bin/kubectl
mv ./kubectl "${HOME}/.local/bin/kubectl"
export PATH="${HOME}/.local/bin/kubectl:$PATH"

########## Get kube-config and install kubectl ##########

echo "$GKE_SERVICE_ACCOUNT_KEY" | base64 -d -i > "${HOME}/gcloud-service-key.json"
gcloud auth activate-service-account --key-file "${HOME}/gcloud-service-key.json"
gcloud container clusters get-credentials travis-test --zone us-central1-c --project cloud-platform-207208

########## Run tests ##########
go test -cover -tags containers_image_storage_stub -v ./...

########## Run integration tests ##########
go test -cover -tags integration,containers_image_storage_stub -v ./...

########## Run e2e tests ##########
go test -cover -tags e2e,containers_image_storage_stub -v ./...
