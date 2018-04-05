package stub

import (
	"fmt"
	"reflect"
	"time"

	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"

	"github.com/coreos/operator-sdk/pkg/sdk/action"
	"github.com/coreos/operator-sdk/pkg/sdk/handler"
	"github.com/coreos/operator-sdk/pkg/sdk/query"
	"github.com/coreos/operator-sdk/pkg/sdk/types"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	//appsv1 "k8s.io/api/apps/v1"
	//"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	//"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewHandler() handler.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx types.Context, event types.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.OneAgent:
		oneagent := o
		updateStatus := false
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "status": oneagent.Status}).Info("received oneagent")

		// query oneagent pods
		podList := getPodList()
		labelSelector := labels.SelectorFromSet(map[string]string{"name": "dynatrace-oneagent"}).String()
		listOps := &metav1.ListOptions{LabelSelector: labelSelector}
		err := query.List("dynatrace", podList, query.WithListOptions(listOps))
		if err != nil {
			logrus.WithFields(logrus.Fields{"pods": podList, "error": err}).Error("failed to query pods")
			return err
		}

		// prepare update status.items
		instances := []v1alpha1.OneAgentInstance{}
		for _, pod := range podList.Items {
			logrus.WithFields(logrus.Fields{"pod": pod.Name, "nodeName": pod.Spec.NodeName}).Debug("processing pod")
			item := v1alpha1.OneAgentInstance{
				PodName:  pod.Name,
				NodeName: pod.Spec.NodeName,
			}
			instances = append(instances, item)
		}
		if !reflect.DeepEqual(instances, oneagent.Status.Items) {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "status.items": instances}).Info("prepare status update")
			updateStatus = true
			oneagent.Status.Items = instances
		}

		// prepare update status.version
		podsToDelete := []corev1.Pod{}
		if oneagent.Status.Version != oneagent.Spec.Version {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "status.version": oneagent.Spec.Version}).Info("prepare status update")
			updateStatus = true
			oneagent.Status.Version = oneagent.Spec.Version

			podsToDelete = podList.Items
		}

		// restart daemonset
		err = deletePods(podsToDelete)
		if err != nil {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err}).Error("failed to delete pods")
			return err
		}

		// update status
		if updateStatus {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "status": oneagent.Status}).Info("updating status")
			oneagent.Status.UpdatedTimestamp = metav1.Now()
			err := action.Update(oneagent)
			if err != nil {
				logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err}).Error("failed to update status")
				return err
			}
		}
	}
	return nil
}

// getPodList returns a v1.PodList object
func getPodList() *corev1.PodList {
	return &corev1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}
}

// deletePods deletes a list of pods
func deletePods(pods []corev1.Pod) error {
	for _, pod := range pods {
		// delete pod
		logrus.WithFields(logrus.Fields{"pod": pod.Name, "nodeName": pod.Spec.NodeName}).Info("deleting pod")
		err := action.Delete(&pod)
		if err != nil {
			logrus.WithFields(logrus.Fields{"pod": pod.Name, "error": err}).Error("failed to delete pod")
			return err
		}

		// wait for pod on node to get "Running" again
		fieldSelector, err := fields.ParseSelector(fmt.Sprintf("spec.nodeName=%v,status.phase=Running,metadata.name!=%v", pod.Spec.NodeName, pod.Name))
		logrus.WithFields(logrus.Fields{"field-selector": fieldSelector}).Debug("waiting for new pod to get ready")
		listOps := &metav1.ListOptions{FieldSelector: fieldSelector.String()}
		items := 0
		for items == 0 {
			time.Sleep(10 * time.Second)
			pList := getPodList()
			err = query.List("dynatrace", pList, query.WithListOptions(listOps))
			if err != nil {
				logrus.WithFields(logrus.Fields{"pods": pList, "error": err}).Error("failed to query pods")
				return err
			}
			items = len(pList.Items)
			if items > 0 {
				time.Sleep(60 * time.Second)
			}
		}
	}
	return nil
}
