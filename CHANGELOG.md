# Changelog

## Future

### Features
* Allow [custom DNS Policy](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy) for OneAgent pods ([#162](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/162))
* Add OpenAPI V3 Schema to CRD objects ([#171](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/171))
* Operator log entries now use ISO-8601 timestamps (e.g., `"2019-10-30T12:59:43.717+0100"`) ([#159](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/159))

### Bug fixes

* _No bug fixes since v0.5.2_

### Other changes
* Most operations now use HTTP Header for authentication with Dynatrace API ([#167](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/167))
* Alpine version for Operator image bumped to 3.10, simplified Dockerfile ([#166](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/166), [#164](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/164))
* Update to nested OLM bundle structure ([#163](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/163))
* Code style improvements ([#158](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/158), [#175](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/175))
* Update to Operator SDK 0.12.0 and Go modules ([#157](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/157), [#172](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/172))
* Using istio.io/client-go to manage Istio objects ([#174](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/174))

## v0.5

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
