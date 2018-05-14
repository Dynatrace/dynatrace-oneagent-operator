#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

DOCKER_REPO_ROOT="/go/src/github.com/Dynatrace/dynatrace-oneagent-operator"
IMAGE=${IMAGE:-"gcr.io/coreos-k8s-scale-testing/codegen:1.9.3"}

docker run --rm \
  -v "$PWD":"$DOCKER_REPO_ROOT" \
  -w "$DOCKER_REPO_ROOT" \
  "$IMAGE" \
  "/go/src/k8s.io/code-generator/generate-internal-groups.sh"  \
  "defaulter" \
  "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/generated" \
  "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis" \
  "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis" \
  "dynatrace:v1alpha1" \
  --go-header-file "$DOCKER_REPO_ROOT/tmp/codegen/boilerplate.go.txt" \
  $@
