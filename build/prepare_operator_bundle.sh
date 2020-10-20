#!/bin/bash

set -eu

cp -r ./deploy/crds/dynatrace* ./deploy/olm/openshift/manifests

if [[ $TRAVIS_BRANCH == "master" ]]; then
  docker build ./deploy/olm/openshift -f ./deploy/olm/openshift/bundle.Dockerfile -t "$OUT_IMAGE"
else
  docker build ./deploy/olm/openshift -f ./deploy/olm/openshift/bundle.Dockerfile -t "$OUT_IMAGE" --label "$LABEL"
fi

docker push "$OUT_IMAGE"

mkdir /tmp/opm_bundle
cd /tmp/opm_bundle
opm index add --bundles "$OUT_IMAGE" --generate

if [[ $TRAVIS_BRANCH == "master" ]]; then
  docker build . -f index.Dockerfile -t "${OUT_IMAGE}"_opm
else
  docker build . -f index.Dockerfile --labels "$LABEL" -t "${OUT_IMAGE}"_opm
fi

docker push "${OUT_IMAGE}"_opm
