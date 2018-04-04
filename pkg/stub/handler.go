package stub

import (
	"reflect"

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
		updateStatus := false
		oneagent := o
		logrus.WithFields(logrus.Fields{"oneagent": o.Name, "status": o.Status}).Info("received oneagent")

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
			logrus.WithFields(logrus.Fields{"pod": pod.Name, "nodeName": pod.Spec.NodeName}).Info("processing pod")
			item := v1alpha1.OneAgentInstance{
				PodName:  pod.Name,
				NodeName: pod.Spec.NodeName,
			}
			instances = append(instances, item)
		}
		if !reflect.DeepEqual(instances, oneagent.Status.Items) {
			logrus.WithFields(logrus.Fields{"oneagent": o.Name, "status.items": instances}).Info("updating status")
			updateStatus = true
			oneagent.Status.Items = instances
		}

		// prepare update status.version
		version := "newVersion"
		if oneagent.Status.Version != version {
			logrus.WithFields(logrus.Fields{"oneagent": o.Name, "status.version": version}).Info("updating status")
			updateStatus = true
			oneagent.Status.Version = version
		}

		// update status
		if updateStatus {
			oneagent.Status.UpdatedTimestamp = metav1.Now()
			err := action.Update(oneagent)
			if err != nil {
				logrus.WithFields(logrus.Fields{"oneagent": o.Name, "error": err}).Error("failed to update status")
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
