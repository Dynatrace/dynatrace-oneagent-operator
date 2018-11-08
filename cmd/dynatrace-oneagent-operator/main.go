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
	group := os.Getenv("MY_POD_GROUP")
	if len(group) == 0 {
		group := "dynatrace.com"
	}
	logrus.Infof("watching namespace: %v", namespace)
	logrus.Infof("Using group: %v", group)
	sdk.Watch(group + "/v1alpha1", "OneAgent", namespace, 120)
	sdk.Handle(stub.NewHandler())
	sdk.Run(context.TODO())
}
