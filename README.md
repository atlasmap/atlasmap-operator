# AtlasMap Operator

[![CircleCI](https://img.shields.io/circleci/project/github/atlasmap/atlasmap-operator/master.svg)](https://circleci.com/gh/atlasmap/atlasmap-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/atlasmap/atlasmap-operator)](https://goreportcard.com/report/github.com/atlasmap/atlasmap-operator)
[![license](https://img.shields.io/github/license/atlasmap/atlasmap-operator.svg)](http://www.apache.org/licenses/LICENSE-2.0)

A Kubernetes operator based on the [Operator SDK](https://github.com/operator-framework/operator-sdk) which can manage [AtlasMap](https://www.atlasmap.io/) on a cluster.

## Custom Resource

```yaml
apiVersion: atlasmap.io/v1alpha1
kind: AtlasMap
metadata:
  name: example-atlasmap
spec:
  # The desired number of replicas
  replicas: 1
  # Override the default AtlasMap image
  image: docker.io/atlasmap/atlasmap
  # The host name to use for the OpenShift route or Kubernetes Ingress. If not specified, this is generated automatically
  routeHostName: example-atlasmap.192.168.42.115.nip.io
  # The amount of CPU to request
  requestCPU: 200m
  # The amount of memory to request
  requestMemory: 256Mi
  # The amount of CPU to limit
  limitCPU: 300m
  # The amount of memory to limit
  limitMemory: 512Mi
```

## Features

The AtlasMap operator can:

### Create
* AtlasMap deployment, route and service objects
### Update
* Reconcile `replicas` count into the deployment
* Reconcile `image` into the deployment to override the [default](https://hub.docker.com/r/atlasmap/atlasmap)
* Reconcile resource requests for CPU and memory into the deployment
* Reconcile resource limits for CPU and memory into the deployment
### Delete
* Remove AtlasMap deployment, route and service objects

## Install

To install the required CRDs, roles, role binding & service account run the following commands as a privileged user.

```console
$ kubectl apply -f deploy/crds/atlasmap.io_atlasmaps_crd.yaml
$ kubectl apply -f deploy/service_account.yaml
$ kubectl apply -f deploy/role.yaml
$ kubectl apply -f deploy/role_binding.yaml
```

To deploy the AtlasMap operator run:

```console
$ kubectl apply -f deploy/operator.yaml
```

Verify that the operator is running:

```console
$ kubectl get deployment atlasmap-operator
NAME                     DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
atlasmap-operator        1         1         1            1           1m
```

If you have `make` installed then you can simplify this by running `make install` & `make deploy`.

## Test

When the operator is running you can deploy an example AtlasMap custom resource:

```console
# Create example-atlasmap
$ kubectl create -f deploy/crds/atlasmap_cr.yaml
atlasmap.atlasmap.io/example-atlasmap created

# Verify example-atlasmap
$ kubectl get atlasmap example-atlasmap
NAME               URL                                                      IMAGE
example-atlasmap   https://example-atlasmap-atlasmap.192.168.42.84.nip.io   docker.io/atlasmap/atlasmap

# Scale example-atlasmap
$ kubectl patch atlasmap example-atlasmap --type='merge' -p '{"spec":{"replicas":3}}'
atlasmap.atlasmap.io/example-atlasmap patched

# Delete example-atlasmap
$ kubectl delete atlasmap example-atlasmap
atlasmap.atlasmap.io "example-atlasmap" deleted
```

## Uninstall

To remove the AtlasMap operator from the cluster run:

```console
$ make uninstall
```

## Development

The AtlasMap operator can be run locally:

```console
$ make run
INFO[0000] Running the operator locally.
```

The AtlasMap operator docker image can be built by running:

```console
$ make build
INFO[0003] Building Docker image docker.io/atlasmap/operator:latest
```

Integration tests can be run by:

```console
$ make test
INFO[0000] Testing operator locally.
```

Or to test a local operator build:

```console
$ make test-local
```
