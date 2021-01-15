module github.com/Dynatrace/dynatrace-oneagent-operator

go 1.15

require (
	github.com/containers/image/v5 v5.8.1
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/go-logr/logr v0.3.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/operator-framework/operator-lib v0.3.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.15.0
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	gotest.tools v2.2.0+incompatible
	istio.io/api v0.0.0-20201125194658-3cee6a1d3ab4
	istio.io/client-go v1.8.0
	k8s.io/api v0.19.4
	k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.4
	sigs.k8s.io/controller-runtime v0.7.0
)
