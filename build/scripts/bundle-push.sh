#!/bin/bash

############################################
# Pushes the AtlasMap OLM bundle to quay.io
############################################

NAMESPACE=${1:-atlasmap}
REPOSITORY=${2:-atlasmap-operator}

MANIFEST_DIR=$(dirname "$0")/../../deploy/olm-catalog/atlasmap-operator
BUNDLE_DIR=/tmp/atlasmap-operator/
BUNDLE_VERSION=$(grep currentCSV ${MANIFEST_DIR}/atlasmap-operator.package.yaml | sed 's/[^0-9\.][v.]*//g')

[ -d ] && rm -rf ${BUNDLE_DIR}
mkdir -p ${BUNDLE_DIR}

cp ${MANIFEST_DIR}/atlasmap-operator.package.yaml ${BUNDLE_DIR}
cp ${MANIFEST_DIR}/${BUNDLE_VERSION}/*.yaml ${BUNDLE_DIR}

if [[ -z "${QUAY_API_TOKEN}" ]]; then
  if [[ -z "$QUAY_USERNAME" ]]; then
      echo -n "Quay Username: "
      read QUAY_USERNAME
  fi

  if [[ -z "$QUAY_PASSWORD" ]]; then
      echo -n "Quay Password: "
      read -s QUAY_PASSWORD
      echo
  fi

  QUAY_API_TOKEN=$(curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '
  {
      "user": {
          "username": "'"${QUAY_USERNAME}"'",
          "password": "'"${QUAY_PASSWORD}"'"
      }
  }' | jq -r '.token')
fi

if which operator-courier > /dev/null; then
  operator-courier --verbose verify --ui_validate_io ${BUNDLE_DIR}
  operator-courier --verbose push ${BUNDLE_DIR} ${NAMESPACE} ${REPOSITORY} ${BUNDLE_VERSION} "${QUAY_API_TOKEN}"
else
  docker run -ti --rm -v ${BUNDLE_DIR}:${BUNDLE_DIR} jamesnetherton/operator-courier:latest \
    operator-courier --verbose verify --ui_validate_io ${BUNDLE_DIR}

  docker run -ti --rm -v ${BUNDLE_DIR}:${BUNDLE_DIR} jamesnetherton/operator-courier:latest \
    operator-courier --verbose push ${BUNDLE_DIR} ${NAMESPACE} ${REPOSITORY} ${BUNDLE_VERSION} "${QUAY_API_TOKEN}"
fi
