#!/bin/bash

set -eu

curl -LO https://github.com/operator-framework/operator-registry/releases/download/v1.14.3/linux-amd64-opm
mv linux-amd64-opm opm
chmod +x opm

cp -r ./deploy/crds/dynatrace* ./deploy/olm/openshift/manifests

if [[ $TRAVIS_BRANCH == "master" ]]; then
  docker build ./deploy/olm/openshift -f ./deploy/olm/openshift/bundle.Dockerfile -t "$OUT_IMAGE"
else
  docker build ./deploy/olm/openshift -f ./deploy/olm/openshift/bundle.Dockerfile -t "$OUT_IMAGE" --label "$LABEL"
fi

docker push "$OUT_IMAGE"

mkdir /tmp/opm_bundle
cd /tmp/opm_bundle
./opm index add --bundles "$OUT_IMAGE" --generate

if [[ $TRAVIS_BRANCH == "master" ]]; then
  docker build . -f index.Dockerfile -t "${OUT_IMAGE}"_opm
else
  docker build . -f index.Dockerfile --labels "$LABEL" -t "${OUT_IMAGE}"_opm
fi

docker push "${OUT_IMAGE}"_opm
