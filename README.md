# dynatrace-oneagent-operator
Kubernetes/OpenShift Operator for managing Dynatrace OneAgent deployments

## Quick Start
Prerequisite: Dynatrace OneAgent is deployed via `DaemonSet` in namespace `dynatrace`.

###### build and push image to registry
```
$ operator-sdk build 172.16.113.133:30000/dynatrace-oneagent-operator
$ docker push 172.16.113.133:30000/dynatrace-oneagent-operator
```
###### setup permissions
Dynatrace OneAgent Operator gets deployed in namespace `operator`. It needs proper permissions to access the namespace `dynatrace`:
```
$ kubectl create -f deploy/rbac.yaml -n dynatrace
$ kubectl patch --type=json -p '[{"op":"add", "path":"/subjects/0/namespace", "value":"operator"}]' -n dynatrace rolebinding/default-account-dynatrace-oneagent-operator
```
###### deploy operator
```
$ kubectl create -f deploy/operator.yaml -n operator
```
###### deploy custom resource
While watching logs from the newly created operator POD, create the custom resource `OneAgent`:
```
kubectl create -f deploy/cr.yaml -n dynatrace
```
## Setup
The project structure was generated via:
```
$ operator-sdk new dynatrace-oneagent-operator --api-version=dynatrace.com/v1alpha1 --kind=OneAgent
```
