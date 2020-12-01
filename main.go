/*


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
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/logger"
	"github.com/Dynatrace/dynatrace-oneagent-operator/version"
	"github.com/prometheus/common/log"
	"github.com/spf13/pflag"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	scheme   = k8sruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

var subcmdCallbacks = map[string]func(ns string, cfg *rest.Config) (manager.Manager, error){
	"operator":             startOperator,
	"webhook-bootstrapper": startWebhookBoostrapper,
	"webhook-server":       startWebhookServer,
}

var errBadSubcmd = errors.New("subcommand must be operator, webhook-bootstrapper, or webhook-server")

var (
	certsDir string
	certFile string
	keyFile  string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(dynatracev1alpha1.AddToScheme(scheme))
	utilruntime.Must(istiov1alpha3.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	webhookServerFlags := pflag.NewFlagSet("webhook-server", pflag.ExitOnError)
	webhookServerFlags.StringVar(&certsDir, "certs-dir", "/mnt/webhook-certs", "Directory to look certificates for.")
	webhookServerFlags.StringVar(&certFile, "cert", "tls.crt", "File name for the public certificate.")
	webhookServerFlags.StringVar(&keyFile, "cert-key", "tls.key", "File name for the private key.")

	pflag.CommandLine.AddFlagSet(webhookServerFlags)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	err := pflag.Set("zap-time-encoding", "iso8601")
	if err != nil {
		log.Error(err, "Failed to set zap-time-encoding")
	}

	logf.SetLogger(logger.NewDTLogger())

	printVersion()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "dynatrace-oneagent-operator-lock",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	subcmd := "operator"
	if args := pflag.Args(); len(args) > 0 {
		subcmd = args[0]
	}

	subcmdFn := subcmdCallbacks[subcmd]
	if subcmdFn == nil {
		log.Error(errBadSubcmd, "Unknown subcommand", "command", subcmd)
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func printVersion() {
	log.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}
