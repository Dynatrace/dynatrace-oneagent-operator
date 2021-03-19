[![TravisCI](https://travis-ci.com/Dynatrace/dynatrace-oneagent-operator.svg)](https://travis-ci.com/Dynatrace/dynatrace-oneagent-operator)
[![Docker Repository on Quay](https://quay.io/repository/dynatrace/dynatrace-oneagent-operator/status "Docker Repository on Quay")](https://quay.io/repository/dynatrace/dynatrace-oneagent-operator)
[![Releases](https://img.shields.io/github/release/Dynatrace/dynatrace-oneagent-operator.svg)](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases)


# Dynatrace OneAgent Operator

This is the home of Dynatrace OneAgent Operator which supports the rollout and lifecycle of [Dynatrace OneAgent](https://www.dynatrace.com/support/help/get-started/introduction/what-is-oneagent/) in Kubernetes and OpenShift clusters.
Rolling out Dynatrace OneAgent via DaemonSet on a cluster is straightforward.
Maintaining its lifecycle places a burden on the operational team.
Dynatrace OneAgent Operator closes this gap by automating the repetitive steps involved in keeping Dynatrace OneAgent at its latest desired version.


## Overview

Dynatrace OneAgent Operator is based on [Operator SDK](https://github.com/operator-framework/operator-sdk) and uses its framework for interacting with Kubernetes and OpenShift environments.
It watches custom resources `OneAgent` and monitors the desired state constantly.
The rollout of Dynatrace OneAgent is managed by a DaemonSet initially.
From here on Dynatrace OneAgent Operator controls the lifecycle and keeps track of new versions and triggers updates if required.

![Overview](./overview.svg)

## Supported platforms

Depending of the version of the Dynatrace OneAgent Operator, it supports the following platforms:

| Dynatrace OneAgent Operator version | Kubernetes | OpenShift Container Platform               |
| ----------------------------------- | ---------- | ------------------------------------------ |
| master                              | 1.18+      | 3.11[<sup>[1]</sup>](#openshift-311), 4.5+ |
| v0.10.0                             | 1.18+      | 3.11[<sup>[1]</sup>](#openshift-311), 4.5+ |
| v0.9.5                              | 1.15+      | 3.11[<sup>[1]</sup>](#openshift-311), 4.3+ |
| v0.8.2                              | 1.14+      | 3.11[<sup>[1]</sup>](#openshift-311), 4.1+ |
| v0.7.1                              | 1.14+      | 3.11[<sup>[1]</sup>](#openshift-311), 4.1+ |
| v0.6.0                              | 1.11+      | 3.11+                                      |
| v0.5.4                              | 1.11+      | 3.11+                                      |
| v0.4.2                              | 1.11+      | 3.11+                                      |
| v0.3.1                              | 1.11-1.15  | 3.11+                                      |
| v0.2.1                              | 1.9-1.15   | 3.9+                                       |

Help topic _How do I deploy Dynatrace OneAgent as a Docker container?_ lists compatible image and OneAgent versions in its [requirements section](https://www.dynatrace.com/support/help/infrastructure/containers/how-do-i-deploy-dynatrace-oneagent-as-docker-container/#requirements).


## Quick Start

The Dynatrace OneAgent Operator acts on its separate namespace `dynatrace`.
It holds the operator deployment and all dependent objects like permissions, custom resources and
corresponding DaemonSets.
Create neccessary objects and observe its logs:

#### Kubernetes
```sh
$ kubectl create namespace dynatrace
$ kubectl apply -f https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/latest/download/kubernetes.yaml
$ kubectl -n dynatrace logs -f deployment/dynatrace-oneagent-operator
```

#### OpenShift
Start by adding a new project as follows:

```sh
$ oc adm new-project --node-selector="" dynatrace
```

If you are installing the Operator on an **OpenShift Container Platform 3.11** environment, in order to use the certified [OneAgent Operator](https://access.redhat.com/containers/#/registry.connect.redhat.com/dynatrace/dynatrace-oneagent-operator) and [OneAgent](https://access.redhat.com/containers/#/registry.connect.redhat.com/dynatrace/oneagent) images from [Red Hat Container Catalog](https://access.redhat.com/containers/) (RHCC), you need to [provide image pull secrets](https://access.redhat.com/documentation/en-us/openshift_container_platform/3.9/html/developer_guide/dev-guide-managing-images#pulling-private-registries-delegated-auth). The Service Accounts on the `openshift.yaml` manifest already have links to the secrets to be created below. Skip this step if you are using OCP 4.x.

```sh
# For OCP 3.11
$ oc -n dynatrace create secret docker-registry redhat-connect --docker-server=registry.connect.redhat.com --docker-username=REDHAT_CONNECT_USERNAME --docker-password=REDHAT_CONNECT_PASSWORD --docker-email=unused
$ oc -n dynatrace create secret docker-registry redhat-connect-sso --docker-server=sso.redhat.com --docker-username=REDHAT_CONNECT_USERNAME --docker-password=REDHAT_CONNECT_PASSWORD --docker-email=unused
```

Finally, for both 4.x and 3.11, we apply the `openshift.yaml` manifest to deploy the Operator:

```sh
$ oc apply -f https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/latest/download/openshift.yaml
$ oc -n dynatrace logs -f deployment/dynatrace-oneagent-operator
```

##### OpenShift 3.11

If using Operator v0.7.0 or greater and OCP 3.11, OCP versions prior to 3.11.188 are suffering of [a bug](https://github.com/openshift/origin/pull/24540) when validating certain CRD objects:

```
The CustomResourceDefinition "oneagents.dynatrace.com" is invalid: spec.validation.openAPIV3Schema: Invalid value: apiextensions.JSONSchemaProps
```

If you have an older OCP 3.11 version then you can remove [this line from openshift.yaml](https://github.com/Dynatrace/dynatrace-oneagent-operator/blob/v0.7.1/deploy/openshift.yaml#L500) (v0.7.1), and deploy it.

#### Create `OneAgent` custom resource for OneAgent rollout
The rollout of Dynatrace OneAgent is governed by a custom resource of type `OneAgent`:
```yaml
apiVersion: dynatrace.com/v1alpha1
kind: OneAgent
metadata:
  # a descriptive name for this object.
  # all created child objects will be based on it.
  name: oneagent
  namespace: dynatrace
spec:
  # dynatrace api url including `/api` path at the end
  apiUrl: https://ENVIRONMENTID.live.dynatrace.com/api
  # disable certificate validation checks for installer download and API communication
  skipCertCheck: false
  # name of secret holding `apiToken` and `paasToken`
  # if unset, name of custom resource is used
  tokens: ""
  # https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/ (optional)
  tolerations:
    - effect: NoSchedule
      key: node-role.kubernetes.io/master
      operator: Exists
```
This is the most basic configuration for the OneAgent object. In case you want to have adjustments please have a look at [our OneAgent Custom Resource example](https://raw.githubusercontent.com/Dynatrace/dynatrace-oneagent-operator/master/config/samples/cr.yaml).
Save this to cr.yaml and apply it to your cluster.
A secret holding tokens for authenticating to the Dynatrace cluster needs to be created upfront.
Create access tokens of type *Dynatrace API* and *Platform as a Service* and use its values in the following commands respectively.
For assistance please refere to [Create user-generated access tokens](https://www.dynatrace.com/support/help/get-started/introduction/why-do-i-need-an-access-token-and-an-environment-id/#create-user-generated-access-tokens).

For Openshift, you can change the image from the default available on Quay.io to the one certified on RHCC by setting `.spec.image` to `registry.connect.redhat.com/dynatrace/oneagent` in the custom resource.

Note: `.spec.tokens` denotes the name of the secret holding access tokens. If not specified OneAgent Operator searches for a secret called like the OneAgent custom resource (`.metadata.name`).

##### Kubernetes
```sh
$ kubectl -n dynatrace create secret generic oneagent --from-literal="apiToken=DYNATRACE_API_TOKEN" --from-literal="paasToken=PLATFORM_AS_A_SERVICE_TOKEN"
$ kubectl apply -f cr.yaml
```

##### OpenShift
```sh
$ oc -n dynatrace create secret generic oneagent --from-literal="apiToken=DYNATRACE_API_TOKEN" --from-literal="paasToken=PLATFORM_AS_A_SERVICE_TOKEN"
$ oc apply -f cr.yaml
```


## Uninstall dynatrace-oneagent-operator
Remove OneAgent custom resources and clean-up all remaining OneAgent Operator specific objects:


#### Kubernetes
```sh
$ kubectl delete -n dynatrace oneagent --all
$ kubectl delete -f https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/latest/download/kubernetes.yaml
```

#### OpenShift
```sh
$ oc delete -n dynatrace oneagent --all
$ oc delete -f https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/latest/download/openshift.yaml
```

## Hacking

See [HACKING](HACKING.md) for details on how to get started enhancing Dynatrace OneAgent Operator.


## Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details on submitting changes.


## License

Dynatrace OneAgent Operator is under Apache 2.0 license. See [LICENSE](LICENSE) for details.
