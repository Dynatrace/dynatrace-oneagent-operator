package istio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	istiov1alpha3 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/istio/v1alpha3"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	istio "istio.io/api/networking/v1alpha3"
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
func BuildServiceEntry(name, host, protocol string, port uint32) *istiov1alpha3.ServiceEntry {
	if net.ParseIP(host) != nil { // It's an IP.
		return buildServiceEntryIP(name, host, port)
	}

	return buildServiceEntryFQDN(name, host, protocol, port)
}

// BuildServiceEntryFQDN returns an Istio ServiceEntry object for the given communication endpoint with a FQDN host.
func buildServiceEntryFQDN(name, host, protocol string, port uint32) *istiov1alpha3.ServiceEntry {

	portStr := strconv.Itoa(int(port))
	protocolStr := strings.ToUpper(protocol)

	spec := istiov1alpha3.ServiceEntrySpec{}
	spec.Hosts = []string{host}
	spec.Location = istio.ServiceEntry_MESH_EXTERNAL
	spec.Ports = []*istio.Port{
		&istio.Port{
			Name:     protocol + "-" + portStr,
			Number:   port,
			Protocol: protocolStr,
		},
	}
	spec.Resolution = istio.ServiceEntry_DNS

	serviceEntry := &istiov1alpha3.ServiceEntry{
		Spec: spec,
	}
	serviceEntry.Name = name
	serviceEntry.Namespace = os.Getenv(k8sutil.WatchNamespaceEnvVar)

	return serviceEntry
}

// BuildServiceEntryIP returns an Istio ServiceEntry object for the given communication endpoint with IP.
func buildServiceEntryIP(name, host string, port uint32) *istiov1alpha3.ServiceEntry {
	portStr := strconv.Itoa(int(port))

	spec := istiov1alpha3.ServiceEntrySpec{}
	spec.Hosts = []string{"ignored.subdomain"}
	spec.Addresses = []string{host + "/32"}
	spec.Location = istio.ServiceEntry_MESH_EXTERNAL
	spec.Ports = []*istio.Port{
		&istio.Port{
			Name:     "TCP-" + portStr,
			Number:   port,
			Protocol: "TCP",
		},
	}
	spec.Resolution = istio.ServiceEntry_NONE

	serviceEntry := &istiov1alpha3.ServiceEntry{
		Spec: spec,
	}
	serviceEntry.Name = name
	serviceEntry.Namespace = os.Getenv(k8sutil.WatchNamespaceEnvVar)

	return serviceEntry
}

// BuildVirtualService returns an Istio VirtualService object for the given communication endpoint.
func BuildVirtualService(name, host, protocol string, port uint32) *istiov1alpha3.VirtualService {
	if net.ParseIP(host) != nil { // It's an IP.
		return nil
	}

	spec := istiov1alpha3.VirtualServiceSpec{}
	spec.Hosts = []string{host}

	switch protocol {
	case "https":
		spec.Tls = []*istio.TLSRoute{
			&istio.TLSRoute{
				Match: []*istio.TLSMatchAttributes{
					&istio.TLSMatchAttributes{
						SniHosts: []string{host},
						Port:     port,
					},
				},
				Route: []*istio.RouteDestination{
					&istio.RouteDestination{
						Destination: &istio.Destination{
							Host: host,
							Port: &istio.PortSelector{
								Port: &istio.PortSelector_Number{Number: port},
							},
						},
					},
				},
			},
		}
	case "http":
		spec.Http = []*istio.HTTPRoute{
			&istio.HTTPRoute{
				Match: []*istio.HTTPMatchRequest{
					&istio.HTTPMatchRequest{
						Port: port,
					},
				},
				Route: []*istio.HTTPRouteDestination{
					&istio.HTTPRouteDestination{
						Destination: &istio.Destination{
							Host: host,
							Port: &istio.PortSelector{
								Port: &istio.PortSelector_Number{Number: port},
							},
						},
					},
				},
			},
		}
	}
	vs := &istiov1alpha3.VirtualService{
		Spec: spec,
	}
	vs.Name = name
	vs.Namespace = os.Getenv(k8sutil.WatchNamespaceEnvVar)

	return vs
}

// BuildNameForEndpoint returns a name to be used as a base to identify Istio objects.
func BuildNameForEndpoint(name string, protocol string, host string, port uint32) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%s-%d", name, protocol, host, port)))
	return hex.EncodeToString(sum[:])
}
