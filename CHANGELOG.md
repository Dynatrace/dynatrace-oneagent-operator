# Changelog

## Future

### Features
* Improve error logging from Dynatrace API requests on Operator ([#185](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/185))
* Allow [custom DNS Policy](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy) for OneAgent pods ([#162](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/162))
* Add OpenAPI V3 Schema to CRD objects ([#171](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/171))
* Operator log entries now use ISO-8601 timestamps (e.g., `"2019-10-30T12:59:43.717+0100"`) ([#159](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/159))
* The service account for pods can now be customized ([#182](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/182), [#187](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/187))
* Custom labels can be added to pods ([#183](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/183), [#191](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/191))
* Validate tokens for OneAgent and show results as conditions on OneAgent status section ([#188](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/188), [#190](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/190))

### Bug fixes
* Operator needs to be restarted after Istio is installed. Fixed on [controller-runtime v0.3.0](https://github.com/kubernetes-sigs/controller-runtime/releases/tag/v0.3.0) ([#172](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/172), [controller-runtime#554](https://github.com/kubernetes-sigs/controller-runtime/pull/554))

### Other changes
* Most operations now use HTTP Header for authentication with Dynatrace API ([#167](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/167))
* Operator Docker images have been merged, and are now based on [UBI](https://www.redhat.com/en/blog/introducing-red-hat-universal-base-image) ([#179](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/179))
* Update to nested OLM bundle structure ([#163](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/163))
* Code style improvements ([#158](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/158), [#175](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/175))
* Update to Operator SDK 0.12.0 and Go modules ([#157](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/157), [#172](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/172))
* Using istio.io/client-go to manage Istio objects ([#174](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/174))

## v0.5

### [v0.5.4](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.5.4)

* Service update to get the latest version of the base image for RedHat Container Catalogue where some vulnerabilities have been fixed.

### [v0.5.3](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.5.3)

* Allow to customize the service account used for creating the OneAgent pods in preparation to release on GKE marketplace ([#182](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/182), [#187](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/187))

### [v0.5.2](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.5.2)

* Fixes: non-default Secret for Dynatrace tokens ignored ([#168](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/168))

### [v0.5.1](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.5.1)

* Fixes: panics when handling node notifications for unmatching OneAgent node selectors ([#160](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/160), [#161](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/161))

### [v0.5.0](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.5.0)

* Better detection and handling of node scaling events. ([#153](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/153), [#154](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/154))
* Improved documentation for installation of the Operator in OCP 3.11 environments. ([#156](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/156))

## Older versions

### [v0.4.2](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.4.2)

* Bug fixes

### [v0.4.1](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.4.1)

* Bug fixes

### [v0.4.0](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.4.0)

* Support automatic configuration of Istio to allow communication from the OneAgent pods to the Dynatrace environment. 
* Added support for Kubernetes 1.16

### [v0.3.1](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.3.1)

* Bug fixes

### [v0.3.0](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.3.0)

* Add configuration field on OneAgent CR to set OneAgent pods' Priority Class.
* Requires Kubernetes 1.11+/OCP 3.11+

### [v0.2.1](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.2.1)

* Add configuration field on OneAgent CR to disable automatic updates from the agent.
* This is the last version supporting Kubernetes 1.9 and 1.10, as well as OCP 3.9 and 3.10.

### [v0.2.0](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.2.0)

* Add configuration field on OneAgent Custom Resource for Node Selector and Resource Quotas of OneAgent pods.

### [v0.1.0](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.1.0)

* Initial release.
* Support for Kubernetes 1.9 and OpenShift Container Platform 3.9 or higher.
