#!/bin/bash

set -eu

cd ./config/olm/openshift
currentVersion="$(ls -d */ | cut -f1 -d'/' | grep -v "current" | sort -rV | head -n 1)"
csv="./${currentVersion}/manifests/dynatrace-monitoring.v${currentVersion}.clusterserviceversion.yaml"
sed -e '/replaces:/ s/^#*/#/' -i "$csv"
bundle_image="./bundle-${currentVersion}.Dockerfile"

if [[ $TRAVIS_BRANCH == "master" ]]; then
  docker build . -f "$bundle_image" -t "$OUT_IMAGE-$ARCH"
else
  docker build . -f "$bundle_image" -t "$OUT_IMAGE-$ARCH" --label "$LABEL"
fi

docker push "$OUT_IMAGE-$ARCH"

mkdir /tmp/opm_bundle
cd /tmp/opm_bundle

if [[ $TRAVIS_CPU_ARCH == "ppc64le" ]]; then
  export OPM_BINARY_IMAGE="registry.redhat.io/openshift4/ose-operator-registry@sha256:c66672f71a6f2863edf9c3291a1e3f2fb529b267e9324064dafba02cef70f159"
  curl -LO https://github.com/operator-framework/operator-registry/releases/download/v1.14.0/linux-ppc64le-opm
  mv linux-ppc64le-opm opm
else
  export OPM_BINARY_IMAGE="registry.redhat.io/openshift4/ose-operator-registry@sha256:9ff790dd2ddc512f369ff70ee272d85cc2ed407e357d0acd45a07e9ecb320d92"
  curl -LO https://github.com/operator-framework/operator-registry/releases/download/v1.14.3/linux-amd64-opm
  mv linux-amd64-opm opm
fi
chmod +x opm

./opm index add --bundles "$OUT_IMAGE-$ARCH" --generate --binary-image "$OPM_BINARY_IMAGE"

if [[ $TRAVIS_BRANCH == "master" ]]; then
  docker build . -f index.Dockerfile -t "${OUT_IMAGE}-$ARCH"_opm
else
  docker build . -f index.Dockerfile --label "$LABEL" -t "${OUT_IMAGE}-$ARCH"_opm
fi

docker push "${OUT_IMAGE}-$ARCH"_opm
