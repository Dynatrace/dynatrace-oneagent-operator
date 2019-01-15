// Package runtime contains the reconciliation logic for the OneAgent Custom Resource.
package runtime

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	api "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"
	rt "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/runtime/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/util"

	"github.com/operator-framework/operator-sdk/pkg/sdk/action"
	"github.com/operator-framework/operator-sdk/pkg/sdk/query"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

// time between consecutive queries for a new pod to get ready
const splayTimeSeconds = uint16(10)

// Reconcile reconciles the OneAgent DaemonSets to the spec specified by
// oneagent custom resource. State changes are detected by comparing spec
// fields from custom resource and its related DaemonSet.
func Reconcile(oneagent *api.OneAgent) error {
	if err := rt.Validate(oneagent); err != nil {
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err}).Error("failed to assert fields")
		return errors.New("failed to assert essential custom resource fields")
	}

	updateStatus := false
	logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "status": oneagent.Status}).Info("received oneagent")

	// default value for .spec.tokens
	if oneagent.Spec.Tokens == "" {
		oneagent.Spec.Tokens = oneagent.Name
		updateStatus = true
	}

	// get access tokens for api authentication
	paasToken, err := getSecretKey(oneagent, "paasToken")
	if err != nil {
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err, "token": "paasToken"}).Error()
		return err
	}
	apiToken, err := getSecretKey(oneagent, "apiToken")
	if err != nil {
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err, "token": "apiToken"}).Error()
		return err
	}

	// element needs to be inserted before it is used in ONEAGENT_INSTALLER_SCRIPT_URL
	if oneagent.Spec.Env[0].Name != "ONEAGENT_INSTALLER_TOKEN" {
		oneagent.Spec.Env = append(oneagent.Spec.Env[:0], append([]corev1.EnvVar{{
			Name: "ONEAGENT_INSTALLER_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: oneagent.Spec.Tokens},
					Key:                  "paasToken"}},
		}}, oneagent.Spec.Env[0:]...)...)
		updateStatus = true
	}

	// create'n'update daemonset
	err = upsertDaemonSet(oneagent)
	if err != nil {
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err}).Error("failed to create or update daemonset")
		return err
	}

	// initialize dynatrace client
	var certificateValidation = dtclient.SkipCertificateValidation(oneagent.Spec.SkipCertCheck)
	dtc, err := dtclient.NewClient(oneagent.Spec.ApiUrl, apiToken, paasToken, certificateValidation)
	if err != nil {
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err}).Warning("failed to get dynatrace rest client")
		return err
	}

	// get desired version
	desired, err := dtc.GetVersionForLatest(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "warning": err}).Warning("failed to get desired version")
		// TODO think about error handling
		// do not return err as it would trigger yet another reconciliation loop immediately
		return nil
	} else if desired != "" && oneagent.Status.Version != desired {
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "previous": oneagent.Status.Version, "desired": desired}).Info("new version available")
		oneagent.Status.Version = desired
		updateStatus = true
	}

	// query oneagent pods
	podList := util.BuildPodList()
	labelSelector := labels.SelectorFromSet(util.BuildLabels(oneagent.Name)).String()
	listOps := &metav1.ListOptions{LabelSelector: labelSelector}
	err = query.List(oneagent.Namespace, podList, query.WithListOptions(listOps))
	if err != nil {
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "pods": podList, "error": err}).Error("failed to query pods")
		return err
	}

	// determine pods to restart
	podsToDelete, instances := rt.GetPodsToRestart(podList.Items, dtc, oneagent)
	if !reflect.DeepEqual(instances, oneagent.Status.Items) {
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "status.items": instances}).Info("status changed")
		updateStatus = true
		oneagent.Status.Items = instances
	}

	// restart daemonset
	err = deletePods(oneagent, podsToDelete)
	if err != nil {
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err}).Error("failed to delete pods")
		return err
	}

	// update status
	if updateStatus {
		oneagent.Status.UpdatedTimestamp = metav1.Now()
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "status": oneagent.Status}).Info("updating status")
		err := action.Update(oneagent)
		if err != nil {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err}).Error("failed to update status")
			return err
		}
	}

	return nil
}

// deletePods deletes a list of pods
//
// Returns an error in the following conditions:
//  - failure on object deletion
//  - timeout on waiting for ready state
func deletePods(cr *api.OneAgent, pods []corev1.Pod) error {
	for _, pod := range pods {
		// delete pod
		logrus.WithFields(logrus.Fields{"oneagent": cr.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName}).Info("deleting pod")
		err := action.Delete(&pod)
		if err != nil {
			logrus.WithFields(logrus.Fields{"oneagent": cr.Name, "pod": pod.Name, "error": err}).Error("failed to delete pod")
			return err
		}

		// wait for pod on node to get "Running" again
		if err := waitPodReadyState(cr, pod); err != nil {
			logrus.WithFields(logrus.Fields{"oneagent": cr.Name, "nodeName": pod.Spec.NodeName, "warning": err}).Warning("timeout waiting on pod to get ready")
			return err
		}
	}

	return nil
}

func waitPodReadyState(cr *api.OneAgent, pod corev1.Pod) error {
	var status error
	fieldSelector, _ := fields.ParseSelector(fmt.Sprintf("spec.nodeName=%v,status.phase=Running,metadata.name!=%v", pod.Spec.NodeName, pod.Name))
	labelSelector := labels.SelectorFromSet(util.BuildLabels(cr.Name))
	logrus.WithFields(logrus.Fields{"field-selector": fieldSelector, "label-selector": labelSelector}).Debug("query pod")
	listOps := &metav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
	for splay := uint16(0); splay < *cr.Spec.WaitReadySeconds; splay += splayTimeSeconds {
		time.Sleep(time.Duration(splayTimeSeconds) * time.Second)
		pList := util.BuildPodList()
		status = query.List(cr.Namespace, pList, query.WithListOptions(listOps))
		if status != nil {
			logrus.WithFields(logrus.Fields{"oneagent": cr.Name, "nodeName": pod.Spec.NodeName, "pods": pList, "warning": status}).Warning("failed to query pods")
			continue
		}
		if n := len(pList.Items); n == 1 && util.GetPodReadyState(&pList.Items[0]) {
			break
		} else if n > 1 {
			status = fmt.Errorf("too many pods found: expected=1 actual=%d", n)
		}
	}
	return status
}

// upsertDaemonSet creates a new DaemonSet object if it does not exist or
// updates an existing one if changes need to be synchronized.
//
// Returns an error in the following conditions:
//  - all k8s apierrors except IsNotFound
//  - failure on daemonset creation
func upsertDaemonSet(oa *api.OneAgent) error {
	ds := util.BuildDaemonSet(oa.Name, oa.Namespace)
	err := query.Get(ds)

	if err == nil {
		// update daemonset
		if rt.HasSpecChanged(&ds.Spec, &oa.Spec) {
			logrus.WithFields(logrus.Fields{"oneagent": oa.Name}).Info("spec changed, updating daemonset")
			rt.ApplyOneAgentSettings(ds, oa.DeepCopy())
			if err := action.Update(ds); err != nil {
				logrus.WithFields(logrus.Fields{"oneagent": oa.Name, "error": err}).Error("failed to update daemonset")
				return err
			}
		}
	} else if apierrors.IsNotFound(err) {
		// create deamonset
		logrus.WithFields(logrus.Fields{"oneagent": oa.Name}).Info("deploying daemonset")
		desiredState := oa.DeepCopy()
		rt.ApplyOneAgentDefaults(ds, desiredState)
		rt.ApplyOneAgentSettings(ds, desiredState)
		err = action.Create(ds)
		if err != nil {
			logrus.WithFields(logrus.Fields{"oneagent": oa.Name, "error": err}).Error("failed to deploy daemonset")
			return err
		}
	} else {
		logrus.WithFields(logrus.Fields{"oneagent": oa.Name, "error": err}).Error("failed to get daemonset")
		return err
	}

	return nil
}

// getSecretKey returns the value of a key from a secret.
//
// Returns an error in the following conditions:
//  - secret not found
//  - key not found
func getSecretKey(cr *api.OneAgent, key string) (string, error) {
	obj := util.BuildSecret(cr.Spec.Tokens, cr.Namespace)

	err := query.Get(obj)
	if err != nil {
		return "", err
	}

	value, ok := obj.Data[key]
	if !ok {
		err = fmt.Errorf("secret %s is missing key %v", cr.Spec.Tokens, key)
		return "", err
	}

	return string(value), nil
}
