package istio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
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
)

// CheckIstioService - Checks if Istio is installed
func CheckIstioService(cfg *rest.Config) error {

	// Creates the dynamic interface.
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	namespace := os.Getenv(k8sutil.WatchNamespaceEnvVar)
	//  List all of the Virtual Services.
	virtualServices, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:   "networking.istio.io",
		Version: "v1alpha3",
	}).Namespace(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	} else if len(virtualServices.Items) == 0 {
		// no error, but no items either
		return fmt.Errorf("no services found with group -  networking.istio.io in namespace %v", namespace)
	}
	return nil
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
