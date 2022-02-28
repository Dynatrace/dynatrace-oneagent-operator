package oneagent

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strconv"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-oneagent-operator/kubesystem"
	"github.com/Dynatrace/dynatrace-oneagent-operator/version"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type daemonSetBuilder struct {
	logger     logr.Logger
	instance   *dynatracev1alpha1.OneAgent
	clusterID  string
	kubeSystem *kubesystem.KubeSystem
}

func newDaemonSetBuilder(logger logr.Logger, instance *dynatracev1alpha1.OneAgent, clusterID string) *daemonSetBuilder {
	return &daemonSetBuilder{
		logger:     logger,
		instance:   instance,
		clusterID:  clusterID,
		kubeSystem: kubesystem.NewKubeSystem(),
	}
}

func (daemonSetBuilder *daemonSetBuilder) newDaemonSetForCR() (*appsv1.DaemonSet, error) {
	instance := daemonSetBuilder.instance
	unprivileged := false

	if ptr := instance.GetOneAgentSpec().UseUnprivilegedMode; ptr != nil {
		unprivileged = *ptr
	}

	podSpec := daemonSetBuilder.newPodSpecForCR(unprivileged)
	selectorLabels := buildLabels(instance.GetName())
	mergedLabels := mergeLabels(instance.GetOneAgentSpec().Labels, selectorLabels)

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.GetName(),
			Namespace:   instance.GetNamespace(),
			Labels:      mergedLabels,
			Annotations: map[string]string{},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: selectorLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: mergedLabels,
					Annotations: map[string]string{
						annotationImageVersion: instance.Status.ImageVersion,
					},
				},
				Spec: podSpec,
			},
		},
	}

	if unprivileged {
		ds.Spec.Template.ObjectMeta.Annotations = map[string]string{
			"container.apparmor.security.beta.kubernetes.io/dynatrace-oneagent": "unconfined",
		}
	}

	dsHash, err := generateDaemonSetHash(ds)
	if err != nil {
		return nil, err
	}
	ds.Annotations[annotationTemplateHash] = dsHash

	return ds, nil
}

func (daemonSetBuilder *daemonSetBuilder) newPodSpecForCR(unprivileged bool) corev1.PodSpec {
	logger := daemonSetBuilder.logger
	instance := daemonSetBuilder.instance
	p := corev1.PodSpec{}

	sa := "dynatrace-oneagent"
	if instance.GetOneAgentSpec().ServiceAccountName != "" {
		sa = instance.GetOneAgentSpec().ServiceAccountName
	} else if unprivileged {
		sa = "dynatrace-oneagent-unprivileged"
	}

	resources := instance.GetOneAgentSpec().Resources
	if resources.Requests == nil {
		resources.Requests = corev1.ResourceList{}
	}
	if _, hasCPUResource := resources.Requests[corev1.ResourceCPU]; !hasCPUResource {
		// Set CPU resource to 1 * 10**(-1) Cores, e.g. 100mC
		resources.Requests[corev1.ResourceCPU] = *resource.NewScaledQuantity(1, -1)
	}

	args := instance.GetOneAgentSpec().Args
	if instance.GetOneAgentSpec().Proxy != nil && (instance.GetOneAgentSpec().Proxy.ValueFrom != "" || instance.GetOneAgentSpec().Proxy.Value != "") {
		args = append(args, "--set-proxy=$(https_proxy)")
	}

	if instance.GetOneAgentSpec().NetworkZone != "" {
		args = append(args, fmt.Sprintf("--set-network-zone=%s", instance.GetOneAgentSpec().NetworkZone))
	}

	if instance.GetOneAgentSpec().WebhookInjection {
		args = append(args, "--set-host-id-source=k8s-node-name")
	}

	args = append(args, "--set-host-property=OperatorVersion="+version.Version)

	// K8s 1.18+ is expected to drop the "beta.kubernetes.io" labels in favor of "kubernetes.io" which was added on K8s 1.14.
	// To support both older and newer K8s versions we use node affinity.

	var secCtx *corev1.SecurityContext
	if unprivileged {
		secCtx = &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{
					"ALL",
				},
				Add: []corev1.Capability{
					"CHOWN",
					"DAC_OVERRIDE",
					"DAC_READ_SEARCH",
					"FOWNER",
					"FSETID",
					"KILL",
					"NET_ADMIN",
					"NET_RAW",
					"SETFCAP",
					"SETGID",
					"SETUID",
					"SYS_ADMIN",
					"SYS_CHROOT",
					"SYS_PTRACE",
					"SYS_RESOURCE",
				},
			},
		}
	} else {
		trueVar := true
		secCtx = &corev1.SecurityContext{
			Privileged: &trueVar,
		}
	}

	p = corev1.PodSpec{
		Containers: []corev1.Container{{
			Args:            args,
			Env:             daemonSetBuilder.prepareEnvVars(),
			Image:           "",
			ImagePullPolicy: corev1.PullAlways,
			Name:            "dynatrace-oneagent",
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"/bin/sh", "-c", "grep -q oneagentwatchdo /proc/[0-9]*/stat",
						},
					},
				},
				InitialDelaySeconds: 30,
				PeriodSeconds:       30,
				TimeoutSeconds:      1,
			},
			Resources:       resources,
			SecurityContext: secCtx,
			VolumeMounts:    daemonSetBuilder.prepareVolumeMounts(),
		}},
		HostNetwork:        true,
		HostPID:            true,
		HostIPC:            true,
		NodeSelector:       instance.GetOneAgentSpec().NodeSelector,
		PriorityClassName:  instance.GetOneAgentSpec().PriorityClassName,
		ServiceAccountName: sa,
		Tolerations:        instance.GetOneAgentSpec().Tolerations,
		DNSPolicy:          instance.GetOneAgentSpec().DNSPolicy,
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "beta.kubernetes.io/arch",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"amd64", "arm64"},
								},
								{
									Key:      "beta.kubernetes.io/os",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"linux"},
								},
							},
						},
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/arch",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"amd64", "arm64"},
								},
								{
									Key:      "kubernetes.io/os",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"linux"},
								},
							},
						},
					},
				},
			},
		},
		Volumes: daemonSetBuilder.prepareVolumes(),
	}

	if instance.GetOneAgentStatus().UseImmutableImage {
		err := daemonSetBuilder.preparePodSpecImmutableImage(&p)
		if err != nil {
			logger.Error(err, "failed to prepare pod spec v2")
		}
	} else {
		err := daemonSetBuilder.preparePodSpecInstaller(&p)
		if err != nil {
			logger.Error(err, "failed to prepare pod spec v1")
		}
	}

	return p
}

func (daemonSetBuilder *daemonSetBuilder) preparePodSpecInstaller(p *corev1.PodSpec) error {
	instance := daemonSetBuilder.instance
	img := oneagentDockerImage

	if instance.GetOneAgentSpec().Image != "" {
		img = instance.GetOneAgentSpec().Image
	} else if daemonSetBuilder.kubeSystem.IsDeployedOnOpenshift {
		img = oneagentRedhatImage
	}

	p.Containers[0].Image = img
	return nil
}

func (daemonSetBuilder *daemonSetBuilder) preparePodSpecImmutableImage(p *corev1.PodSpec) error {
	instance := daemonSetBuilder.instance
	pullSecretName := instance.GetName() + "-pull-secret"

	if instance.GetOneAgentSpec().CustomPullSecret != "" {
		pullSecretName = instance.GetOneAgentSpec().CustomPullSecret
	}

	p.ImagePullSecrets = append(p.ImagePullSecrets, corev1.LocalObjectReference{
		Name: pullSecretName,
	})

	if instance.Spec.Image != "" {
		p.Containers[0].Image = instance.Spec.Image
		return nil
	}

	i, err := utils.BuildOneAgentImage(instance.GetSpec().APIURL, instance.GetOneAgentSpec().AgentVersion)
	if err != nil {
		return err
	}
	p.Containers[0].Image = i

	return nil
}

func (daemonSetBuilder *daemonSetBuilder) prepareVolumes() []corev1.Volume {
	instance := daemonSetBuilder.instance
	volumes := []corev1.Volume{
		{
			Name: "host-root",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/",
				},
			},
		},
	}

	if instance.GetOneAgentSpec().TrustedCAs != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "certs",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: instance.GetOneAgentSpec().TrustedCAs,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "certs",
							Path: "certs.pem",
						},
					},
				},
			},
		})
	}

	return volumes
}

func (daemonSetBuilder *daemonSetBuilder) prepareVolumeMounts() []corev1.VolumeMount {
	instance := daemonSetBuilder.instance
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "host-root",
			MountPath: "/mnt/root",
		},
	}

	if instance.GetOneAgentSpec().TrustedCAs != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "certs",
			MountPath: "/mnt/dynatrace/certs",
		})
	}

	return volumeMounts
}

func (daemonSetBuilder *daemonSetBuilder) prepareEnvVars() []corev1.EnvVar {
	instance := daemonSetBuilder.instance
	clusterID := daemonSetBuilder.clusterID
	type reservedEnvVar struct {
		Name    string
		Default func(ev *corev1.EnvVar)
		Value   *corev1.EnvVar
	}

	reserved := []reservedEnvVar{
		{
			Name: "DT_K8S_NODE_NAME",
			Default: func(ev *corev1.EnvVar) {
				ev.ValueFrom = &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}
			},
		},
		{
			Name: "DT_K8S_CLUSTER_ID",
			Default: func(ev *corev1.EnvVar) {
				ev.Value = clusterID
			},
		},
	}

	if instance.Spec.WebhookInjection {
		reserved = append(reserved,
			reservedEnvVar{
				Name: "ONEAGENT_DISABLE_CONTAINER_INJECTION",
				Default: func(ev *corev1.EnvVar) {
					ev.Value = "true"
				},
			})
	}

	if !instance.GetStatus().UseImmutableImage {
		reserved = append(reserved,
			reservedEnvVar{
				Name: "ONEAGENT_INSTALLER_TOKEN",
				Default: func(ev *corev1.EnvVar) {
					ev.ValueFrom = &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: utils.GetTokensName(instance)},
							Key:                  utils.DynatracePaasToken,
						},
					}
				},
			},
			reservedEnvVar{
				Name: "ONEAGENT_INSTALLER_SCRIPT_URL",
				Default: func(ev *corev1.EnvVar) {
					ev.Value = fmt.Sprintf("%s/v1/deployment/installer/agent/unix/default/latest?Api-Token=$(ONEAGENT_INSTALLER_TOKEN)&arch=x86&flavor=default", instance.GetOneAgentSpec().APIURL)
				},
			},
			reservedEnvVar{
				Name: "ONEAGENT_INSTALLER_SKIP_CERT_CHECK",
				Default: func(ev *corev1.EnvVar) {
					ev.Value = strconv.FormatBool(instance.GetOneAgentSpec().SkipCertCheck)
				},
			})
	}

	if p := instance.GetOneAgentSpec().Proxy; p != nil && (p.Value != "" || p.ValueFrom != "") {
		reserved = append(reserved, reservedEnvVar{
			Name: "https_proxy",
			Default: func(ev *corev1.EnvVar) {
				if p.ValueFrom != "" {
					ev.ValueFrom = &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: p.ValueFrom},
							Key:                  "proxy",
						},
					}
				} else {
					ev.Value = p.Value
				}
			},
		})
	}

	reservedMap := map[string]*reservedEnvVar{}
	for i := range reserved {
		reservedMap[reserved[i].Name] = &reserved[i]
	}

	// Split defined environment variables between those reserved and the rest

	instanceEnv := instance.GetOneAgentSpec().Env

	var remaining []corev1.EnvVar
	for i := range instanceEnv {
		if p := reservedMap[instanceEnv[i].Name]; p != nil {
			p.Value = &instanceEnv[i]
			continue
		}
		remaining = append(remaining, instanceEnv[i])
	}

	// Add reserved environment variables in that order, and generate a default if unset.

	var env []corev1.EnvVar
	for i := range reserved {
		ev := reserved[i].Value
		if ev == nil {
			ev = &corev1.EnvVar{Name: reserved[i].Name}
			reserved[i].Default(ev)
		}
		env = append(env, *ev)
	}

	return append(env, remaining...)
}

func hasDaemonSetChanged(a, b *appsv1.DaemonSet) bool {
	return getTemplateHash(a) != getTemplateHash(b)
}

func generateDaemonSetHash(ds *appsv1.DaemonSet) (string, error) {
	data, err := json.Marshal(ds)
	if err != nil {
		return "", err
	}

	hasher := fnv.New32()
	_, err = hasher.Write(data)
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(uint64(hasher.Sum32()), 10), nil
}

func getTemplateHash(a metav1.Object) string {
	if annotations := a.GetAnnotations(); annotations != nil {
		return annotations[annotationTemplateHash]
	}
	return ""
}
