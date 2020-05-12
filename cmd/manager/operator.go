/*
Copyright 2020 Dynatrace LLC.

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

package main

import (
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/namespace"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/nodes"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/oneagent"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/oneagentapm"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func startOperator(ns string, cfg *rest.Config) (manager.Manager, error) {
	mgr, err := manager.New(cfg, manager.Options{Namespace: ns})
	if err != nil {
		return nil, err
	}

	log.Info("Registering Components.")

	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	for _, f := range []func(manager.Manager) error{
		oneagent.Add,
		oneagentapm.Add,
		namespace.Add,
		nodes.Add,

		// To add once we start supporting the OneAgentIM CRD.
		// oneagentim.Add,
	} {
		if err := f(mgr); err != nil {
			return nil, err
		}
	}

	return mgr, nil
}
