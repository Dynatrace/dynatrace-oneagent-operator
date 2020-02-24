#!/bin/bash

set -eu

# Get the latest operator-sdk
OPERATOR_SDK="/usr/local/bin/operator-sdk"
LATEST_OPERATOR_SDK_RELEASE=$(curl -s https://api.github.com/repos/operator-framework/operator-sdk/releases/latest | grep tag_name | cut -d '"' -f 4)
if [ ! -f "/usr/local/bin/operator-sdk" ]; then
    curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${LATEST_OPERATOR_SDK_RELEASE}/operator-sdk-${LATEST_OPERATOR_SDK_RELEASE}-x86_64-linux-gnu
    chmod +x operator-sdk-${LATEST_OPERATOR_SDK_RELEASE}-x86_64-linux-gnu && sudo mkdir -p /usr/local/bin/ && sudo cp operator-sdk-${LATEST_OPERATOR_SDK_RELEASE}-x86_64-linux-gnu /usr/local/bin/operator-sdk && rm operator-sdk-${LATEST_OPERATOR_SDK_RELEASE}-x86_64-linux-gnu
fi

LATEST_OPERATOR_RELEASE=$(curl -s https://api.github.com/repos/dynatrace/dynatrace-oneagent-operator/releases/latest | grep tag_name | cut -d '"' -f 4 | awk '{print substr($1,2); }')

mkdir -p ./deploy/olm-catalog
mkdir -p ./deploy/olm/kubernetes/${VERSION_TAG}
mkdir -p ./deploy/olm/openshift/${VERSION_TAG}

# Copy over the latest existing version of the CSV for K8s, generate the CSV and move it back to the K8s folder
cp -r ./deploy/olm/kubernetes/${LATEST_OPERATOR_RELEASE} ./deploy/olm-catalog/dynatrace-monitoring/
$OPERATOR_SDK generate csv --csv-channel alpha --csv-version $VERSION_TAG --csv-config=./deploy/olm/config_k8s.yaml --from-version $LATEST_OPERATOR_RELEASE --operator-name dynatrace-monitoring
mv ./deploy/olm-catalog/dynatrace-monitoring/${VERSION_TAG} ./deploy/olm/kubernetes/
rm -rf ./deploy/olm-catalog/dynatrace-monitoring/${LATEST_OPERATOR_RELEASE}

# Copy over the latest existing version of the CSV for OCP, generate the CSV and move it back to the OCP folder
cp -r ./deploy/olm/openshift/${LATEST_OPERATOR_RELEASE} ./deploy/olm-catalog/dynatrace-monitoring/
$OPERATOR_SDK generate csv --csv-channel alpha --csv-version $VERSION_TAG --csv-config=./deploy/olm/config_ocp.yaml --from-version $LATEST_OPERATOR_RELEASE --operator-name dynatrace-monitoring
mv ./deploy/olm-catalog/dynatrace-monitoring/${VERSION_TAG} ./deploy/olm/openshift/
rm -rf ./deploy/olm-catalog/dynatrace-monitoring/${LATEST_OPERATOR_RELEASE}

# Remove the created folder
rm -rf ./deploy/olm-catalog/

# Copy CRDs to new CSV folders
cp ./deploy/crds/dynatrace.com_oneagents_crd.yaml ./deploy/olm/kubernetes/${VERSION_TAG}/oneagents.dynatrace.com.crd.yaml
cp ./deploy/crds/dynatrace.com_oneagents_crd.yaml ./deploy/olm/openshift/${VERSION_TAG}/oneagents.dynatrace.com.crd.yaml

# Prepare files in a separate branch and push them to github
echo -n $GITHUB_KEY | base64 -d > ~/.ssh/id_rsa

git clone git@github.com:Dynatrace/dynatrace-oneagent-operator.git
cd ./dynatrace-oneagent-operator
git config user.email "cloudplatform@dynatrace.com"
git config user.name "Dynatrace Bot"

git checkout -b "csv/${VERSION_TAG}"
git add .
git commit -m "New CSV file for version ${VERSION_TAG}"
git push origin "csv/${VERSION_TAG}"
