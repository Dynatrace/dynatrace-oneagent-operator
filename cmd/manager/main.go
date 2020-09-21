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
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/logger"
	"github.com/Dynatrace/dynatrace-oneagent-operator/version"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var log = logf.Log.WithName("cmd")

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

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
	log.Info(fmt.Sprintf("Version of dynatrace-oneagent-operator: %v", version.Version))
}

func main() {
	webhookServerFlags := pflag.NewFlagSet("webhook-server", pflag.ExitOnError)
	webhookServerFlags.StringVar(&certsDir, "certs-dir", "/mnt/webhook-certs", "Directory to look certificates for.")
	webhookServerFlags.StringVar(&certFile, "cert", "tls.crt", "File name for the public certificate.")
	webhookServerFlags.StringVar(&keyFile, "cert-key", "tls.key", "File name for the private key.")

	pflag.CommandLine.AddFlagSet(webhookServerFlags)
	pflag.CommandLine.AddFlagSet(zap.FlagSet())
	err := pflag.Set("zap-time-encoding", "iso8601")
	if err != nil {
		log.Error(err, "Failed to set zap-time-encoding")
	}
	pflag.Parse()

	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(logger.NewDTLogger())

	printVersion()

	subcmd := "operator"
	if args := pflag.Args(); len(args) > 0 {
		subcmd = args[0]
	}

	subcmdFn := subcmdCallbacks[subcmd]
	if subcmdFn == nil {
		log.Error(errBadSubcmd, "Unknown subcommand", "command", subcmd)
		os.Exit(1)
	}

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Become the leader before proceeding
	err = leader.Become(context.Background(), fmt.Sprintf("dynatrace-oneagent-%s-lock", subcmd))
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	mgr, err := subcmdFn(namespace, cfg)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
