
ORG = atlasmap
NAMESPACE ?= atlasmap
PROJECT = atlasmap-operator
TAG = latest
OPERATOR_SDK_VERSION=v0.7.0

.PHONY: compile
compile:
	go build -o=atlasmap-operator ./cmd/manager/main.go

.PHONY: generate
generate:
	operator-sdk generate k8s

.PHONY: build
build:
	operator-sdk build docker.io/${ORG}/${PROJECT}:${TAG}

.PHONY: image
image: compile build

.PHONY: install
install: install-crds
	kubectl apply -f deploy/service_account.yaml -n ${NAMESPACE}
	kubectl apply -f deploy/role.yaml -n ${NAMESPACE}
	kubectl apply -f deploy/role_binding.yaml -n ${NAMESPACE}

.PHONY: install-crds
install-crds:
	kubectl apply -f deploy/crds/atlasmap_v1alpha1_atlasmap_crd.yaml

.PHONY: uninstall
uninstall:
	kubectl delete -f deploy/crds/atlasmap_v1alpha1_atlasmap_crd.yaml
	kubectl delete -f deploy/service_account.yaml -n ${NAMESPACE}
	kubectl delete -f deploy/role.yaml -n ${NAMESPACE}
	kubectl delete -f deploy/role_binding.yaml -n ${NAMESPACE}

.PHONY: deploy
deploy:
	kubectl apply -f deploy/operator.yaml -n ${NAMESPACE}

.PHONY: test-local
test-local:
	operator-sdk test local ./test/e2e --go-test-flags "-v" --namespace ${NAMESPACE} --up-local

.PHONY: test
test:
	operator-sdk test local ./test/e2e --go-test-flags "-v" --namespace ${NAMESPACE}

.PHONY: run
run:
	operator-sdk up local --namespace=${NAMESPACE} --operator-flags=""

.PHONY: install-operator-sdk
install-operator-sdk:
	curl -Lo operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_VERSION}/operator-sdk-${OPERATOR_SDK_VERSION}-x86_64-linux-gnu && chmod +x operator-sdk && sudo mv operator-sdk /usr/local/bin/
