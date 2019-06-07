#!/bin/bash

REGISTRY_NAMESPACE=${1:-atlasmap}

if kubectl get operatorsource atlasmap -n openshift-marketplace > /dev/null; then
    kubectl delete operatorsource atlasmap -n openshift-marketplace > /dev/null
    sleep 1
fi

NAMESPACE=marketplace
if kubectl get namespace openshift > /dev/null; then
    NAMESPACE=openshift-marketplace
fi

cat <<EOF | kubectl create -f -
apiVersion: operators.coreos.com/v1
kind: OperatorSource
metadata:
  name: atlasmap
  namespace: ${NAMESPACE}
spec:
  type: appregistry
  endpoint: https://quay.io/cnr
  registryNamespace: ${REGISTRY_NAMESPACE}
  displayName: "AtlasMap Operators"
  publisher: "AtlasMap"
EOF
