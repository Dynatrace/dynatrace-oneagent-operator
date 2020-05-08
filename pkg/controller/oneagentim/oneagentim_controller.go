/*
Copyright YEAR Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package oneagentim

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/oneagent"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/utils"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new OneAgentIM Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, oneagent.NewOneAgentReconciler(
		mgr.GetClient(),
		mgr.GetAPIReader(),
		mgr.GetScheme(),
		mgr.GetConfig(),
		log.Log.WithName("oneagentim.controller"),
		utils.BuildDynatraceClient,
		&dynatracev1alpha1.OneAgentIM{}))
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *oneagent.ReconcileOneAgent) error {
	// Create a new controller
	c, err := controller.New("oneagentim-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource OneAgentIM
	err = c.Watch(&source.Kind{Type: &dynatracev1alpha1.OneAgentIM{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource DaemonSets and requeue the owner OneAgent
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &dynatracev1alpha1.OneAgentIM{},
	})
	if err != nil {
		return err
	}

	return nil
}
