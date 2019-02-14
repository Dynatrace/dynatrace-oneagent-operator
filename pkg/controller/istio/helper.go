package istio

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"strconv"
	"strings"
)

var (
	VirtualServiceGVK = schema.GroupVersionKind{
		Group:   "networking.istio.io",
		Version: "v1alpha3",
		Kind:    "VirtualService",
	}

	ServiceEntryGVK = schema.GroupVersionKind{
		Group:   "networking.istio.io",
		Version: "v1alpha3",
		Kind:    "ServiceEntry",
	}
)

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
