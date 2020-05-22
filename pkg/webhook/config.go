package webhook

const (
	// LabelInstance can be set in a Namespace and indicates the corresponding OneAgent object assigned to it.
	LabelInstance = "oneagent.dynatrace.com/instance"

	// AnnotationInject can be set at pod or namespace label to enable/disable injection, where at pod level has higher
	// priority.
	AnnotationInject = "oneagent.dynatrace.com/inject"

	// AnnotationInjected is set to "true" by the webhook to Pods to indicate that it has been modified.
	AnnotationInjected = "oneagent.dynatrace.com/injected"

	// AnnotationFlavor can be set on a Pod to configure which code modules flavor to download. It's set to "default"
	// if not set.
	AnnotationFlavor = "oneagent.dynatrace.com/flavor"

	// AnnotationTechnologies can be set on a Pod to configure which code module technologies to download. It's set to
	// "all" if not set.
	AnnotationTechnologies = "oneagent.dynatrace.com/technologies"

	// AnnotationInstallPath can be set on a Pod to configure on which directory the OneAgent will be available from,
	// defaults to DefaultInstallPath if not set.
	AnnotationInstallPath = "oneagent.dynatrace.com/install-path"

	// AnnotationInstallerUrl can be set on a Pod to configure the installer url for downloading the agent
	// defaults to the PaaS installer download url of your tenant
	AnnotationInstallerUrl = "oneagent.dynatrace.com/installer-url"

	// DefaultInstallPath is the default directory to install the app-only OneAgent package.
	DefaultInstallPath = "/opt/dynatrace/oneagent-paas"

	// SecretConfigName is the name of the secret where the Operator replicates the config data.
	SecretConfigName = "dynatrace-oneagent-config"

	// SecretCertsName is the name of the secret where the webhook certificates are stored.
	SecretCertsName = "dynatrace-oneagent-webhook-certs"

	// ServiceName is the name used for the webhook's corresponding Service and MutatingWebhookConfiguration objects.
	ServiceName = "dynatrace-oneagent-webhook"
)
