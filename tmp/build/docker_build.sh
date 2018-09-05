#!/usr/bin/env bash

if ! which docker > /dev/null; then
	echo "docker needs to be installed"
	exit 1
fi

: ${IMAGE:?"Need to set IMAGE, e.g. gcr.io/<repo>/<your>-operator"}
: ${DOCKERFILE:="tmp/build/Dockerfile"}

echo "building container ${IMAGE}..."
docker build -t "${IMAGE}" -f "${DOCKERFILE}" .
