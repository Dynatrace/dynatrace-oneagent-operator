#!/bin/bash

set -eu

VERSION=$(echo $TRAVIS_TAG | sed 's/v//')

# Get the latest operator-sdk
OPERATOR_SDK="/usr/local/bin/operator-sdk"

if [ ! -f "/usr/local/bin/operator-sdk" ]; then
    LATEST_OPERATOR_SDK_RELEASE=$(curl -s https://api.github.com/repos/operator-framework/operator-sdk/releases/latest | grep tag_name | cut -d '"' -f 4)
    curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${LATEST_OPERATOR_SDK_RELEASE}/operator-sdk-${LATEST_OPERATOR_SDK_RELEASE}-x86_64-linux-gnu
    chmod +x operator-sdk-${LATEST_OPERATOR_SDK_RELEASE}-x86_64-linux-gnu
    sudo mkdir -p /usr/local/bin/
    sudo mv operator-sdk-${LATEST_OPERATOR_SDK_RELEASE}-x86_64-linux-gnu /usr/local/bin/operator-sdk
fi

LATEST_OPERATOR_RELEASE=$(ls -d ./deploy/olm/kubernetes/*/ | sort -r | head -n 1 | xargs -n 1 basename)

mkdir -p ./deploy/olm-catalog/dynatrace-monitoring/${LATEST_OPERATOR_RELEASE}
mkdir -p ./deploy/olm/kubernetes/${VERSION}
mkdir -p ./deploy/olm/openshift/${VERSION}

# Copy over the latest existing version of the CSV for K8s, generate the CSV and move it back to the K8s folder
cp -r ./deploy/olm/kubernetes/${LATEST_OPERATOR_RELEASE} ./deploy/olm-catalog/dynatrace-monitoring/
$OPERATOR_SDK generate csv --csv-channel alpha --csv-version $VERSION --csv-config=./deploy/olm/config_k8s.yaml --from-version $LATEST_OPERATOR_RELEASE --operator-name dynatrace-monitoring
sed -i "i/dynatrace-oneagent-operator:v${LATEST_OPERATOR_RELEASE}/dynatrace-oneagent-operator:v${VERSION}" ./deploy/olm-catalog/dynatrace-monitoring/${VERSION}
mv ./deploy/olm-catalog/dynatrace-monitoring/${VERSION} ./deploy/olm/kubernetes/
rm -rf ./deploy/olm-catalog/dynatrace-monitoring/${LATEST_OPERATOR_RELEASE}

# Copy over the latest existing version of the CSV for OCP, generate the CSV and move it back to the OCP folder
cp -r ./deploy/olm/openshift/${LATEST_OPERATOR_RELEASE} ./deploy/olm-catalog/dynatrace-monitoring/
$OPERATOR_SDK generate csv --csv-channel alpha --csv-version $VERSION --csv-config=./deploy/olm/config_ocp.yaml --from-version $LATEST_OPERATOR_RELEASE --operator-name dynatrace-monitoring
sed -i "i/dynatrace-oneagent-operator:v${LATEST_OPERATOR_RELEASE}/dynatrace-oneagent-operator:v${VERSION}" ./deploy/olm-catalog/dynatrace-monitoring/${VERSION}
mv ./deploy/olm-catalog/dynatrace-monitoring/${VERSION} ./deploy/olm/openshift/
rm -rf ./deploy/olm-catalog/dynatrace-monitoring/${LATEST_OPERATOR_RELEASE}

# Remove the created folder
rm -rf ./deploy/olm-catalog/

# Copy CRDs to new CSV folders
cp ./deploy/crds/dynatrace.com_oneagents_crd.yaml ./deploy/olm/kubernetes/${VERSION}/oneagents.dynatrace.com.crd.yaml
cp ./deploy/crds/dynatrace.com_oneagents_crd.yaml ./deploy/olm/openshift/${VERSION}/oneagents.dynatrace.com.crd.yaml

# Prepare files in a separate branch and push them to github
echo -n $GITHUB_KEY | base64 -d > ~/.ssh/id_rsa
chmod 400 ~/.ssh/id_rsa

cd /tmp
git clone git@github.com:Dynatrace/dynatrace-oneagent-operator.git
cd ./dynatrace-oneagent-operator

cp -r $TRAVIS_BUILD_DIR/deploy/olm/kubernetes/$VERSION ./deploy/olm/kubernetes/
cp -r $TRAVIS_BUILD_DIR/deploy/olm/openshift/$VERSION ./deploy/olm/openshift/
cat $TRAVIS_BUILD_DIR/deploy/olm/openshift/oneagent.package.yaml | sed "s/${LATEST_OPERATOR_RELEASE}/${VERSION}/" > ./deploy/olm/openshift/oneagent.package.yaml
cat $TRAVIS_BUILD_DIR/deploy/olm/kubernetes/oneagent.package.yaml | sed "s/${LATEST_OPERATOR_RELEASE}/${VERSION}/" > ./deploy/olm/kubernetes/oneagent.package.yaml

git config user.email "cloudplatform@dynatrace.com"
git config user.name "Dynatrace Bot"

git checkout -b "csv/${TRAVIS_TAG}"
git add .
git commit -m "New CSV file for version ${TRAVIS_TAG}"
git push --set-upstream origin "csv/${TRAVIS_TAG}"
