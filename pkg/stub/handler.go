package stub

import (
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/oneagent"

	"github.com/operator-framework/operator-sdk/pkg/sdk/handler"
	"github.com/operator-framework/operator-sdk/pkg/sdk/types"
)

func NewHandler() handler.Handler {
	return &Handler{}
}

type Handler struct{}

func (h *Handler) Handle(ctx types.Context, event types.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.OneAgent:
		// Ignore the delete event since the garbage collector will clean up
		// all secondary resources for the CR via OwnerReference
		if event.Deleted {
			return nil
		}
		return oneagent.Reconcile(o)
	}
	return nil
}
