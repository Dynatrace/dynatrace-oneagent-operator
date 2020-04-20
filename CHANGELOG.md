# Changelog

### Future

#### Features
* Implement webhook to inject the OneAgent in App-only mode ([#234](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/234))
  * This feature can be enabled by setting the label `oneagent.dynatrace.com/instance: <oneagent-object-name>` on the namespaces to monitor.

## v0.7

### [v0.7.1](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.7.1)

### Bug fixes
* Marked for Termination events are not a point in time instead of a time range of a few minutes ([#229](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/229))
* Fixed error message when OneAgent has been already removed from the cache but the node was still there ([#232](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/232))

### Other changes
* Added environment variable 'RELATED_IMAGE_DYNATRACE_ONEAGENT' as preparation for RedHat marketplace release ([#228](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/228))
* Fixed some problems with the current Travis CI build ([#230](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/230))

### [v0.7.0](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.7.0)

#### Breaking changes
* This version drops support for Kubernetes 1.11, 1.12, and 1.13 ([#219](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/219), [#220](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/220))

#### Features
* Separated the logic for watching the nodes into nodes_controller to handle scaling correctly ([#189](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/189), [#196](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/196))
* Show operator phase in the `status.phase` field of the OneAgent object ([#197](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/197))
* Build ARM64 images for the Operator ([#201](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/201), [#211](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/211))
* No longer change the OneAgent .spec section to set defaults ([#206](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/206))
* Added a setting to configure a proxy via the CR ([#207](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/207))
* Added a setting to add custom CA certificates via the CR - These changes are only done for the Operator image as of now and the changes in the OneAgent image are in progress ([#208](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/208))
* Added proper error handling for Dynatrace API quota limit ([#216](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/216))

#### Bug fixes
* Handle sporadic (and benign) race conditions where the error below would appear ([#194](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/194)),
  ```
  Operation cannot be fulfilled on oneagents.dynatrace.com \"oneagent\": the object has been modified; please apply your changes to the latest version and try again
  ```
* Proxy environment variables (e.g., `http_proxy`, etc.) can be ignored on Operator container when `skipCertCheck` is true ([#204](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/204))
* Istio objects don't have an owner object, so wouldn't get removed if the OneAgent object is deleted ([#217](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/217))

#### Other changes
* As part of the support for ARM ([#201](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/201), [#203](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/203))
  * Migrate CI/CD workflow from CircleCI to TravisCI
  * Development snapshot images are now being published to Docker Hub
* Support deprecation of `beta.kubernetes.io/arch` and `beta.kubernetes.io/os` labels ([#199](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/199))
* Update to Operator SDK 0.15.1 ([#200](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/200))
* Initial work to ease release automation ([#198](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/198))
* Added automatic creation of CSV file for OLM ([#210](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/210))
* Now Marked for Termination events will be sent only for deleted Nodes ([#213](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/213), [#214](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/214), [#223](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/223))
* Use `v1` instead of `v1beta1` for `rbac.authorization.k8s.io` objects ([#215](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/215))
* Add OLM manifests for v0.7.0 release ([#226](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/226))

## v0.6

### [v0.6.0](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.6.0)

#### Features
* Improve error logging from Dynatrace API requests on Operator ([#185](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/185))
* Allow [custom DNS Policy](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy) for OneAgent pods ([#162](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/162))
* Add OpenAPI V3 Schema to CRD objects ([#171](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/171))
* Operator log entries now use ISO-8601 timestamps (e.g., `"2019-10-30T12:59:43.717+0100"`) ([#159](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/159))
* The service account for pods can now be customized ([#182](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/182), [#187](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/187))
* Custom labels can be added to pods ([#183](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/183), [#191](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/191))
* Validate tokens for OneAgent and show results as conditions on OneAgent status section ([#188](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/188), [#190](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/190))

#### Bug fixes
* Operator needs to be restarted after Istio is installed. Fixed on [controller-runtime v0.3.0](https://github.com/kubernetes-sigs/controller-runtime/releases/tag/v0.3.0) ([#172](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/172), [controller-runtime#554](https://github.com/kubernetes-sigs/controller-runtime/pull/554))

#### Other changes
* Installation steps on Readme are now for stable releases ([#205](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/205))
* Most operations now use HTTP Header for authentication with Dynatrace API ([#167](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/167))
* Operator Docker images have been merged, and are now based on [UBI](https://www.redhat.com/en/blog/introducing-red-hat-universal-base-image) ([#179](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/179))
* Update to nested OLM bundle structure ([#163](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/163))
* Code style improvements ([#158](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/158), [#175](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/175))
* Update to Operator SDK 0.12.0 and Go modules ([#157](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/157), [#172](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/172))
* Using istio.io/client-go to manage Istio objects ([#174](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/174))
* Add OLM manifests for v0.6.0 ([#193](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/193))

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
