# Changelog

### Future

## v0.10

### v0.10.0

#### Bug fixes
* Don't look at the cluster version when deploying the OneAgent using immutable images. Under certain conditions this may stop the Operator from deploying the OneAgent at all ([#376](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/376))
* Upgrade OneAgent Pods using the immutable image by looking at the version label embedded on the images ([#376](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/376))

#### Other changes
* Upgrade to Operator SDK 1.3 ([#351](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/351))
  * Use ConfigMaps for leases ([#367](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/367))
* Fix version and deployment instructions for dev builds ([#379](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/379))

## v0.9

### v0.9.5

#### Bug fixes
* Adapted the update interval to only do the request very 15 minutes ([#368](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/368))

### v0.9.4

#### Bug fixes
* Temporary files for the injection via webhook now get placed on an emptyDir to support readonlyFileSystems ([#353](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/353))
* ImagePullPolicy for install-oneagent container used by OneAgentAPM got changed to `Always` to allow updates ([#358](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/358))
* Nodes in a non-working state will not get added as empty entries to the init.sh script anymore ([#359](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/359))
* Querying for the Dynatrace cluster version will just be done per OneAgent resource when using the immutable image ([#360](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/360))

### v0.9.3

* Minor release just for Helm

### v0.9.2

#### Features
* Added property to change default C standard library ([#341](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/341))

#### Bug fixes
* Don't restart OneAgent Pods if OneAgent version has been downgraded and those Pods were already restarted ([#339](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/339), [#345](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/345))

#### Other changes
* Updated the UBI-minimal base image from version 8.2 to 8.3 ([#344](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/344))

### v0.9.1
* Minor release just for OperatorHub

### v0.9.0

#### Features
* Provide Prometheus metrics for the Operator pod on port 8080, and Webhook Pod on ports 8383 and 8484 ([#305](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/305))
* Control whether the init container crashes in case of download failures through the `oneagent.dynatrace.com/failure-policy: fail` Pod annotation, off by default ([#288](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/288))
  * Regardless of the annotation, if the unzip operation fails, a file `package.zip` will be included on the target directory for debugging purposes.

* Resource limits and requests for the OneAgentAPM initContainer are configurable on the `.spec.resources` field ([#332](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/332))

* Early Adopter: support full-stack OneAgent running on unprivileged mode ([#324](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/324), [#333](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/333))

#### Bug fixes
* Logged errors when API token is missing on OneAgentAPM's secret ([#298](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/298))
* Fixed printing the name of the used token secret for OneAgent instances ([311](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/311))
* Fixed setting instances metadata when auto-update is disabled ([#313](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/313))
* Fixed logging problem - [incorrect stackdriver severity on GCP](https://github.com/Dynatrace/dynatrace-oneagent-operator/issues/277) ([#318](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/318))

#### Other changes
* Added support for immutable OneAgent images - waiting for support on Dynatrace cluster ([#300](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/300), [#286](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/286), [#290](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/290), [#301](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/301))
* Added check if cluster and agent versions are compatible with immutable images ([#314](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/314), [#334](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/334))
* Immutable image mode is disabled when a custom installer URL annotation is set, or `.spec.useImmutableImage` is false ([#306](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/306))
* Pod and node metadata added for the OneAgent ([#294](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/294), [#295](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/295), [#308](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/308), [#325](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/325), [#326](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/326))
* Code cleanup to remove unused functions, variables and beautify the code ([#302](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/302))
* Sped up TravisCI duration ([#310](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/310), [#312](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/312))
* Upgrade to Go 1.15 ([#310](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/310))
* Add linter to TravisCI pipeline ([#316](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/316))
* App-only init container will log an warning when the full-stack OneAgent has been injected on it ([#323](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/323))
* Improve error message when OneAgentAPM is missing ([#327](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/327))
* Improve descriptions on cr.yaml example ([#328](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/328))

## v0.8

### [v0.8.2](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.8.2)

#### Bug fixes
* Reworked update mechanism to prevent downgrades ([#320](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/320))

### [v0.8.1](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.8.1)

#### Features
* Publish Operator stable images also to Quay ([#304](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/304))

#### Bug fixes
* Update status of OneAgentAPM if token is missing ([#285](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/285), [#287](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/287))
* Marked for termination events are now sent when a node is deleted, or when it's cordoned, and then periodically after each hour while in that state ([#279](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/279), [#303](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/303))
* Operator was updating the DaemonSet even if there were no changes ([#289](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/289))
* Certificates secret not updated on renewal, causing renewals every 5 minutes ([#297](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/297))
* Add OLM manifests for v0.8.1 ([#307](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/307))

### [v0.8.0](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.8.0)

#### Features
* Implement webhook to inject the OneAgent in App-only mode ([#234](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/234), [#237](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/237), [#239](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/239), [#250](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/250))
  * This feature can be enabled by setting the label `oneagent.dynatrace.com/instance: <oneagent-object-name>` on the namespaces to monitor.
  * CA and server certificates are generated for the webhook by the Operator, and renewed automatically after 365 and 7 days, respectively ([#244](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/244))
  * OneAgent app-only package and logs will be stored on `/opt/dynatrace/oneagent-paas` inside the containers by default. It can be configured with the `oneagent.dynatrace.com/install-path` annotation on Pods ([#251](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/251))
  * OneAgent app-only package will be downloaded from the provided tenant by default. It can be configured with the `oneagent.dynatrace.com/installer-url` annotation on Pods ([#258](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/258))
  * Certificates location can be configured on the webhook server with the `--certs-dir`, `--cert`, and `--cert-key` command line arguments ([#261](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/261))
  * When setting a custom installer url the authentication header won't be sent ([#264](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/264))
* Added a setting to configure a NetworkZone via the CR ([#270](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/270))

#### Bug fixes
* Phase now gets set to 'Deploying' while the OneAgent gets updated ([#267](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/267))

#### Other changes
* Removed kubernetes.yaml and openshift.yaml from master and generate them with kustomize instead ([#238](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/238), [#254](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/254))
* Updated the Go version from 1.13 to 1.14 ([#242](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/242))
* Updated the Operator SDK version from 0.15.0 to 0.17.0 ([#243](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/243))
* The different operator and webhook modes are encapsulated in a single binary ([#252](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/252), [#253](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/253))
* Webhook's init container only downloads 64bits package ([#256](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/256))
* Include Service and MutatingWebhookConfiguration objects in manifests ([#262](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/262), [#266](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/266))
* Upgrade base image to ubi-minimal:8.2 ([#255](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/255))
* Include Operator version as a custom property for hosts ([#212](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/212))
* Ignore hosts that haven't seen in the last 30 minutes when looking for hosts ([#271](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/271), [~~#257~~](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/257))
* Adjust permissions for the webhook ([#263](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/263))
* Refactor workflow from OneAgent controller ([#268](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/268))
* Automatically update conditions if migrating from earlier Operator versions ([#269](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/269))
* Remove unused metadata from webhook-injected Pods ([#272](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/272))
* Changes in preparation for v0.8.0 release ([#273](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/273), [#274](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/274))
* Add OLM manifests for v0.8.0 ([#275](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/275))

## v0.7

### [v0.7.1](https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/tag/v0.7.1)

#### Bug fixes
* Marked for Termination events are not a point in time instead of a time range of a few minutes ([#229](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/229))
* Fixed error message when OneAgent has been already removed from the cache but the node was still there ([#232](https://github.com/Dynatrace/dynatrace-oneagent-operator/pull/232))

#### Other changes
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
