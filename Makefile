ATLASMAP_IMAGE=docker.io/atlasmap/atlasmap
ATLASMAP_IMAGE_TAG=latest
GIT_COMMIT=$(shell git rev-parse --short HEAD || echo 'Unknown')
NAMESPACE ?= atlasmap
OPERATOR_SDK_VERSION=v0.15.1
ORG = atlasmap
PROJECT = atlasmap-operator
QUAY_NAMESPACE ?= atlasmap
QUAY_REPOSITORY ?= atlasmap-operator
ROOT_PACKAGE := $(shell go list ./version)
TAG = latest
VERSION = $(shell grep Version version/version.go | cut -d \" -f2)

.PHONY: compile
compile:
	go build -ldflags "-X $(ROOT_PACKAGE).GitCommit=$(GIT_COMMIT)" -o=atlasmap-operator ./cmd/manager/main.go

.PHONY: generate
generate:
	operator-sdk generate k8s
	operator-sdk generate crds

.PHONY: generate-config
generate-config:
	build/scripts/generate-source.sh $(ATLASMAP_IMAGE) $(ATLASMAP_IMAGE_TAG)

.PHONY: generate-csv
generate-csv:
	operator-sdk generate csv --csv-version $(VERSION) --update-crds --from-version $(shell cat deploy/olm-catalog/atlasmap-operator/atlasmap-operator.package.yaml | grep currentCSV | cut -f2 -d'v')

.PHONY: build
build: generate-config
	operator-sdk build --go-build-args "-ldflags -X=$(ROOT_PACKAGE).GitCommit=$(GIT_COMMIT)" docker.io/${ORG}/${PROJECT}:${TAG}

.PHONY: image
image: compile build

.PHONY: install
install: install-crds
	kubectl apply -f deploy/service_account.yaml -n ${NAMESPACE}
	kubectl apply -f deploy/role.yaml -n ${NAMESPACE}
	kubectl apply -f deploy/role_binding.yaml -n ${NAMESPACE}
	kubectl apply -f deploy/cluster_role.yaml
	kubectl apply -f deploy/cluster_role_binding.yaml

.PHONY: install-crds
install-crds:
	kubectl apply -f deploy/crds/atlasmaps.atlasmap.io.crd.yaml

.PHONY: uninstall
uninstall:
	kubectl delete -f deploy/crds/atlasmaps.atlasmap.io.crd.yaml
	kubectl delete -f deploy/service_account.yaml -n ${NAMESPACE}
	kubectl delete -f deploy/role.yaml -n ${NAMESPACE}
	kubectl delete -f deploy/role_binding.yaml -n ${NAMESPACE}
	kubectl delete -f deploy/cluster_role.yaml
	kubectl delete -f deploy/cluster_role_binding.yaml

.PHONY: deploy
deploy:
	kubectl apply -f deploy/operator.yaml -n ${NAMESPACE}

.PHONY: test-local
test-local:
	operator-sdk test local ./test/e2e --go-test-flags "-v" --namespace ${NAMESPACE} --up-local

.PHONY: test
test:
	go test -v $(shell go list ./... | grep -v e2e)
	operator-sdk test local ./test/e2e --go-test-flags "-v" --namespace ${NAMESPACE}

.PHONY: run
run:
	operator-sdk run --local --namespace=${NAMESPACE} --operator-flags=""

.PHONY: scorecard
scorecard:
	operator-sdk scorecard

.PHONY: install-operator-sdk
install-operator-sdk:
	curl -Lo operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_VERSION}/operator-sdk-${OPERATOR_SDK_VERSION}-x86_64-linux-gnu && chmod +x operator-sdk && sudo mv operator-sdk /usr/local/bin/

.PHONY: olm-bundle-push
olm-bundle-push:
	build/scripts/bundle-push.sh $(QUAY_NAMESPACE) $(QUAY_REPOSITORY)

.PHONY: olm-operator-source
olm-operator-source:
	build/scripts/create-operator-source.sh $(QUAY_NAMESPACE)
