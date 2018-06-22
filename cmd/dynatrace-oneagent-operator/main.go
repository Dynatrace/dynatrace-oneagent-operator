package main

import (
	"context"
	"os"
	"runtime"

	stub "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/stub"
	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	"github.com/sirupsen/logrus"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()
	namespace := os.Getenv("MY_POD_NAMESPACE")
	logrus.Infof("watching namespace: %v", namespace)
	sdk.Watch("dynatrace.com/v1alpha1", "OneAgent", namespace, 120)
	sdk.Handle(stub.NewHandler())
	sdk.Run(context.TODO())
}
