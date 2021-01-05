module github.com/Dynatrace/dynatrace-oneagent-operator

go 1.14

require (
	cloud.google.com/go v0.57.0 // indirect
	github.com/containers/image/v5 v5.5.2
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/google/go-cmp v0.4.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/onsi/gomega v1.9.0 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/operator-framework/operator-sdk v0.17.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	golang.org/x/sys v0.0.0-20200523222454-059865788121 // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	golang.org/x/tools v0.0.0-20200804011535-6c149bb5ef0d // indirect
	google.golang.org/genproto v0.0.0-20200527145253-8367513e4ece // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	gotest.tools v2.2.0+incompatible
	honnef.co/go/tools v0.0.1-2020.1.5 // indirect
	istio.io/api v0.0.0-20191109011911-e51134872853
	istio.io/client-go v0.0.0-20191119175647-1aefa51f7583
	k8s.io/api v0.17.4
	k8s.io/apiextensions-apiserver v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)
