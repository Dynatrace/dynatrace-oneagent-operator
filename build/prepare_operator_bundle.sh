#!/bin/bash

set -eu

bundle_image="./config/olm/openshift/bundle-current.Dockerfile"

set -x
grep --version
currentVersion="$(ls -t ./config/olm/openshift | grep -E "^(\d+\.)?(\d+\.)?(\*|\d+)$" | head -n 1)"
csv="./config/olm/openshift/${currentVersion}/manifests/dynatrace-monitoring.v${currentVersion}.clusterserviceversion.yaml"
sed -e '/replaces:/ s/^#*/#/' -i "$csv"

set +x

if [[ $TRAVIS_BRANCH == "master" ]]; then
  docker build ./config/olm/openshift -f "$bundle_image" -t "$OUT_IMAGE"
else
  docker build ./config/olm/openshift -f "$bundle_image" -t "$OUT_IMAGE" --label "$LABEL"
fi

docker push "$OUT_IMAGE"

mkdir /tmp/opm_bundle
cd /tmp/opm_bundle

curl -LO https://github.com/operator-framework/operator-registry/releases/download/v1.14.3/linux-amd64-opm
mv linux-amd64-opm opm
chmod +x opm

./opm index add --bundles "$OUT_IMAGE" --generate

if [[ $TRAVIS_BRANCH == "master" ]]; then
  docker build . -f index.Dockerfile -t "${OUT_IMAGE}"_opm
else
  docker build . -f index.Dockerfile --label "$LABEL" -t "${OUT_IMAGE}"_opm
fi

docker push "${OUT_IMAGE}"_opm

sed -i '/replaces:/s/^#//g' "$csv"
