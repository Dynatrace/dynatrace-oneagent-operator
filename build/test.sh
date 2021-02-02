#!/bin/bash

########## Prepare directories for Kubebuilder ##########
sudo mkdir -p /usr/local/kubebuilder
sudo chmod 777 /usr/local/kubebuilder

########## Fetch Kubebuilder ##########
if [ ! -d "/usr/local/kubebuilder/bin" ]; then
  curl -L https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.3.1/kubebuilder_2.3.1_linux_amd64.tar.gz -o kubebuilder.tar.gz
  tar -zxvf kubebuilder.tar.gz --strip-components=1 -C /usr/local/kubebuilder
fi

########## Get Kube-Config ##########

echo "$GCLOUD_SERVICE_KEY_DEV" | base64 -d -i > ${HOME}/gcloud-service-key.json
gcloud auth activate-service-account --key-file ${HOME}/gcloud-service-key.json
gcloud container clusters get-credentials travis-test --zone us-central1-c --project cloud-platform-207208

########## Run tests ##########
go test -cover -tags containers_image_storage_stub -v ./...

########## Run integration tests ##########
go test -cover -tags integration,containers_image_storage_stub -v ./...
