# Dynatrace OneAgent Operator

This is the home of Dynatrace OneAgent Operator which supports the rollout and lifecycle of [Dynatrace OneAgent](https://www.dynatrace.com/support/help/get-started/introduction/what-is-oneagent/) in Kubernetes and OpenShift clusters.
Rolling out Dynatrace OneAgent via DaemonSet on a cluster is straightforward.
Maintaining its lifecycle places a burden on the operational team.
Dynatrace OneAgent Operator closes this gap by automating the repetitive steps involved in keeping Dynatrace OneAgent at its latest desired version.


## Overview

Dynatrace OneAgent Operator is based on [Operator SDK](https://github.com/coreos/operator-sdk) and uses its framework for interacting with Kubernetes and OpenShift environments.
It watches custom resources `OneAgent` and monitors the desired state constantly.
The rollout of Dynatrace OneAgent is managed by a DaemonSet initially.
From here on Dynatrace OneAgent Operator controlls the lifecycle and keeps track of new versions and triggers updates if required.

![Overview](./overview.svg)

## Usage

##### Create namespace and setup permissions

The Dynatrace OneAgent Operator acts on its separate namespace `dynatrace`.
It holds the operator deployment and all dependent objects like permissions, custom resources and
corresponding DaemonSets.
```
$ kubectl create -f https://raw.githubusercontent.com/Dynatrace/dynatrace-oneagent-operator/master/deploy/namespace.yaml
$ kubectl create -f https://raw.githubusercontent.com/Dynatrace/dynatrace-oneagent-operator/master/deploy/rbac.yaml
```

##### Deploy dynatrace-oneagent-operator
```
$ kubectl create -f https://raw.githubusercontent.com/Dynatrace/dynatrace-oneagent-operator/master/deploy/crd.yaml
$ kubectl create -f https://raw.githubusercontent.com/Dynatrace/dynatrace-oneagent-operator/master/deploy/operator.yaml
```
The activity of Dynatrace OneAgent Operator can be observed by following its logs:
```
$ kubectl -n dynatrace logs -f deployment/dynatrace-oneagent-operator
```

##### Create `OneAgent` custom resource for OneAgent rollout
The rollout of Dynatrace OneAgent is governed by a custom resource of type `OneAgent`:
```yaml
apiVersion: "dynatrace.com/v1alpha1"
kind: "OneAgent"
metadata:
  # a descriptive name for this object.
  # all created child objects will be based on it.
  name: "example"
  namespace: "dynatrace"
spec:
  # dynatrace api url including `/api` path at the end
  apiUrl: "https://ENVIRONMENTID.live.dynatrace.com/api"
  # dynatrace api token: `/#settings/integration/apikeys`
  apiToken: ""
  # dynatrace paas token (aka installer token): `/#settings/integration/paastokens`
  paasToken: ""
  # node selector to control the selection of nodes (optional)
  nodeSelector: {}
```
Save the snippet to a file or use [./deploy/cr.yaml](https://raw.githubusercontent.com/Dynatrace/dynatrace-oneagent-operator/master/deploy/cr.yaml) from this repository and adjust its values accordingly. Create the custom resource:
```
$ kubectl create -f cr.yaml
```
The status of the Dynatrace OneAgent rollout can be observed by watching the pod list:
```
$ kubectl -n dynatrace get pods --selector=dynatrace=oneagent,oneagent -w -o wide
```


## Hack on Dynatrace OneAgent Operator

[Operator SDK](https://github.com/coreos/operator-sdk) is the underlying framework for Dynatrace
OneAgent Operator. The `operator-sdk` tool needs to be installed upfront as outlined in the
[Operator SDK User Guide](https://github.com/coreos/operator-sdk/blob/master/doc/user-guide.md#install-the-operator-sdk-cli).

##### Build and push your image
Replace `REGISTRY` with your Registry\`s URN:
```
$ cd $GOPATH/src/github.com/Dynatrace/dynatrace-oneagent-operator
$ operator-sdk build REGISTRY/dynatrace-oneagent-operator
$ docker push REGISTRY/dynatrace-oneagent-operator
```

##### Deploy operator
Change the `image` field in `./deploy/operator.yaml` to the URN of your image.
Apart from that follow the instructions in the usage section above.


## How to Contribute

You are welcome to contribute to Dynatrace OneAgent Operator.
If you have improvements to Dynatrace OneAgent Operator, please submit your pull request.
For those just getting started, consult this  [guide](https://help.github.com/articles/creating-a-pull-request-from-a-fork/).

Please note we have a [code of conduct](./CODEOFCONDUCT.md), please follow it in all your interactions with the project.
