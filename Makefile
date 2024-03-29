# Current Operator version
VERSION ?= 0.0.1
# Default bundle image tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true, crdVersions=v1"
CRD_OPTIONS_OCP311 ?= "crd:trivialVersions=true, preserveUnknownFields=false, crdVersions=v1beta1"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
ENVTEST_ASSETS_DIR = $(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.3/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -ldflags="-X 'github.com/Dynatrace/dynatrace-oneagent-operator/version.Version=${TAG}'" -tags containers_image_storage_stub -o ./build/_output/bin/dynatrace-oneagent-operator ./

# Run against the configured Kubernetes cluster in ~/.kube/config
run: export RUN_LOCAL=true
run: export POD_NAMESPACE=dynatrace
run: export DEBUG_OPERATOR=true
run: generate fmt vet manifests
	go run ./

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/kubernetes && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/kubernetes | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/default/bases

manifests-ocp311: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS_OCP311) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/ocp311/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG} --build-arg GO_BUILD_ARGS="-linkmode external -extldflags '-static' -s -w"

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests kustomize
	operator-sdk generate kustomize manifests -q
	cd config/olm/$(PLATFORM) && $(KUSTOMIZE) edit set image "docker.io/dynatrace/dynatrace-oneagent-operator:snapshot"="$(IMG)"
	$(KUSTOMIZE) build config/olm/$(PLATFORM) | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle
	rm -rf ./config/olm/$(PLATFORM)/$(VERSION)
	mkdir -p ./config/olm/$(PLATFORM)/$(VERSION)
	mv ./bundle/* ./config/olm/$(PLATFORM)/$(VERSION)
	mv ./config/olm/$(PLATFORM)/$(VERSION)/manifests/dynatrace-monitoring.clusterserviceversion.yaml ./config/olm/$(PLATFORM)/$(VERSION)/manifests/dynatrace-monitoring.v$(VERSION).clusterserviceversion.yaml

	grep -v "COPY bundle/manifests /manifests/" bundle.Dockerfile > tmp.Dockerfile
	grep -v "COPY bundle/metadata /metadata/" tmp.Dockerfile > bundle.Dockerfile
	grep -v "COPY bundle/tests/scorecard /tests/scorecard/" bundle.Dockerfile > tmp.Dockerfile

	echo "COPY config/olm/$(PLATFORM)/$(VERSION)/manifests /manifests/" >> tmp.Dockerfile
	echo "COPY config/olm/$(PLATFORM)/$(VERSION)/metadata /metadata/" >> tmp.Dockerfile

	mv tmp.Dockerfile bundle.Dockerfile
	mv ./bundle.Dockerfile ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile -t $(BUNDLE_IMG) .

bundle-push:
	docker push $(BUNDLE_IMG)
