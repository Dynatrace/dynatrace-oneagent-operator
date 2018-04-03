package stub

import (
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"

	//"github.com/coreos/operator-sdk/pkg/sdk/action"
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
		logrus.Infof("received oneagent: %v", o.Name)

		// query oneagent pods
		podList := getPodList()
		labelSelector := labels.SelectorFromSet(map[string]string{"name": "dynatrace-oneagent"}).String()
		listOps := &metav1.ListOptions{LabelSelector: labelSelector}
		err := query.List("dynatrace", podList, query.WithListOptions(listOps))
		if err != nil {
			logrus.Errorf("failed to query pods %v: %v", podList, err)
			return err
		}

		// do something
		for _, pod := range podList.Items {
			logrus.Infof("processing pod %v on %v", pod.Name, pod.Spec.NodeName)
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
