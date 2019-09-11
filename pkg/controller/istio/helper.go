package istio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

var (
	istioGVRName = "networking.istio.io"

	// VirtualServiceGVK => definition of virtual service GVK for oneagent
	VirtualServiceGVK = schema.GroupVersionKind{
		Group:   istioGVRName,
		Version: "v1alpha3",
		Kind:    "VirtualService",
	}

	// ServiceEntryGVK => definition of virtual service GVK for oneagent
	ServiceEntryGVK = schema.GroupVersionKind{
		Group:   istioGVRName,
		Version: "v1alpha3",
		Kind:    "ServiceEntry",
	}
)

// CheckIstioEnabled checks if Istio is installed
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

// BuildServiceEntry returns an Istio ServiceEntry object for the given communication endpoint.
func BuildServiceEntry(name string, host string, port uint32, protocol string) []byte {
	if net.ParseIP(host) != nil { // It's an IP.
		return BuildServiceEntryIP(name, host, port)
	}

	return BuildServiceEntryFQDN(name, host, port, protocol)
}

// BuildServiceEntryFQDN returns an Istio ServiceEntry object for the given communication endpoint with a FQDN host.
func BuildServiceEntryFQDN(name string, host string, port uint32, protocol string) []byte {
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

// BuildServiceEntryIP returns an Istio ServiceEntry object for the given communication endpoint with IP.
func BuildServiceEntryIP(name string, host string, port uint32) []byte {
	portStr := strconv.Itoa(int(port))

	return []byte(`{
    "apiVersion": "networking.istio.io/v1alpha3",
    "kind": "ServiceEntry",
    "metadata": {
        "name": "` + name + `",
        "namespace": "` + os.Getenv(k8sutil.WatchNamespaceEnvVar) + `"
    },
    "spec": {
        "hosts": [ "ignored.subdomain" ],
        "addresses": [ "` + host + `/32" ],
        "location": "MESH_EXTERNAL",
        "ports": [{
            "name": "TCP-` + portStr + `",
            "number": ` + portStr + `,
            "protocol": "TCP"
        }],
        "resolution": "NONE"
    }
}`)
}

// BuildVirtualService returns an Istio VirtualService object for the given communication endpoint.
func BuildVirtualService(name string, host string, port uint32, protocol string) []byte {
	if net.ParseIP(host) != nil { // It's an IP.
		return nil
	}

	switch protocol {
	case "https":
		return buildVirtualServiceHTTPS(name, host, port)
	case "http":
		return buildVirtualServiceHTTP(name, host, port)
	}

	return nil
}

func buildVirtualServiceHTTPS(name string, host string, port uint32) []byte {
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

func buildVirtualServiceHTTP(name string, host string, port uint32) []byte {
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
                "port": ` + portStr + `
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

// BuildNameForEndpoint returns a name to be used as a base to identify Istio objects.
func BuildNameForEndpoint(name string, protocol string, host string, port uint32) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%s-%d", name, protocol, host, port)))
	return hex.EncodeToString(sum[:])
}
