#!/bin/bash

if [[ "$GCR" == "true" ]]; then
    echo "$GCLOUD_SERVICE_KEY" | base64 -d | gcloud auth activate-service-account --key-file=-
    gcloud --quiet config set project "$GCP_PROJECT"
    gcloud auth configure-docker
else
    TAG=$TAG-$TRAVIS_CPU_ARCH
fi

if [[ -z "$LABEL" ]]; then
    docker build . -f ./build/Dockerfile -t "$IMAGE:$TAG"
else
    docker build . -f ./build/Dockerfile -t "$IMAGE:$TAG" --label "$LABEL"
fi

echo "Pushing docker image"
docker push "$IMAGE:$TAG"
