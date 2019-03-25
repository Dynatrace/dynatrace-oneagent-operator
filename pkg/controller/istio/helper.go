package istio

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

var (
	// VirtualServiceGVK => definition of virtual service GVK for oneagent
	VirtualServiceGVK = schema.GroupVersionKind{
		Group:   "networking.istio.io",
		Version: "v1alpha3",
		Kind:    "VirtualService",
	}

	// ServiceEntryGVK => definition of virtual service GVK for oneagent
	ServiceEntryGVK = schema.GroupVersionKind{
		Group:   "networking.istio.io",
		Version: "v1alpha3",
		Kind:    "ServiceEntry",
	}

	istioGVRName = "networking.istio.io"
)

// CheckIstioEnabled - Checks if Istio is installed
func CheckIstioEnabled(cfg *rest.Config) (bool, error) {
	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return false, err
	}
	apiGroupList, err := client.ServerGroups()
	if err != nil {
		return false, err
	}

	for _, apiGroup := range apiGroupList.Groups {
		if apiGroup.Name == istioGVRName {
			return true, nil
		}
	}
	return false, nil
}

func BuildServiceEntry(name string, host string, port uint32, protocol string) []byte {
	portStr := strconv.Itoa(int(port))
	protocolStr := strings.ToUpper(protocol)

	return []byte(`{
    "apiVersion": "networking.istio.io/v1alpha3",
    "kind": "ServiceEntry",
    "metadata": {
        "name": "` + name + `",
		"namespace": "` + os.Getenv(k8sutil.WatchNamespaceEnvVar) + `"
    },
    "spec": {
        "hosts": [ "` + host + `" ],
        "location": "MESH_EXTERNAL",
        "ports": [{
				"name": "` + protocol + portStr + `",
                "number": ` + portStr + `,
                "protocol": "` + protocolStr + `"
		}],
        "resolution": "DNS"
    }
}`)
}

func BuildVirtualService(name string, host string, port uint32, protocol string) []byte {
	switch protocol {
	case "https":
		return buildVirtualServiceHttps(name, host, port)
	case "http":
		return buildVirtualServiceHttp(name, host, port)
	}

	return []byte(`{}`)
}

func buildVirtualServiceHttps(name string, host string, port uint32) []byte {
	portStr := strconv.Itoa(int(port))

	return []byte(`{
    "apiVersion": "networking.istio.io/v1alpha3",
    "kind": "VirtualService",
    "metadata": {
        "name": "` + name + `",
		"namespace": "` + os.Getenv(k8sutil.WatchNamespaceEnvVar) + `"
    },
    "spec": {
        "hosts": [ "` + host + `" ],
        "tls": [{
            "match": [{
            	"port": ` + portStr + `,
                "sni_hosts": [ "` + host + `" ]
			}],
            "route": [{
				"destination": {
					"host": "` + host + `",
					"port": { "number": ` + portStr + ` }
				}
			}]
		}]
    }
}`)
}

func buildVirtualServiceHttp(name string, host string, port uint32) []byte {
	portStr := strconv.Itoa(int(port))

	return []byte(`{
    "apiVersion": "networking.istio.io/v1alpha3",
    "kind": "VirtualService",
    "metadata": {
        "name": "` + name + `",
		"namespace": "` + os.Getenv(k8sutil.WatchNamespaceEnvVar) + `"
    },
    "spec": {
        "hosts": [ "` + host + `" ],
        "http": [{
            "match": [{
            	"port": ` + portStr + `,
                "headers": [{ "Host": "` + host + `" }]
			}],
            "route": [{
				"destination": {
					"host": "` + host + `",
					"port": { "number": ` + portStr + ` }
				}
			}]
		}]
    }
}`)
}

func BuildNameForEndpoint(name string, host string, port uint32) string {
	portStr := strconv.Itoa(int(port))
	src := make([]byte, len(name)+len(host)+len(portStr))
	src = strconv.AppendQuote(src, name)
	src = strconv.AppendQuote(src, host)
	src = strconv.AppendQuote(src, portStr)

	sum := sha256.Sum256(src)

	return hex.EncodeToString(sum[:])
}
