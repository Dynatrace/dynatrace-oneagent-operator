package dtclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Client is the interface for the Dynatrace REST API client.
type Client interface {
	// GetLatestAgentVersion gets the latest agent version for the given OS and installer type.
	// Returns the version as received from the server on success.
	//
	// Returns an error for the following conditions:
	//  - os or installerType is empty
	//  - IO error or unexpected response
	//  - error response from the server (e.g. authentication failure)
	//  - the agent version is not set or empty
	GetLatestAgentVersion(os, installerType string) (string, error)

	// GetAgentVersionForIP returns the agent version running on the host with the given IP address.
	// Returns the version string formatted as "Major.Minor.Revision.Timestamp" on success.
	//
	// Returns an error for the following conditions:
	//  - the IP is empty
	//  - IO error or unexpected response
	//  - error response from the server (e.g. authentication failure)
	//  - a host with the given IP cannot be found
	//  - the agent version for the host is not set
	//
	// The list of all hosts with their IP addresses is cached the first time this method is called. Use a new
	// client instance to fetch a new list from the server.
	GetAgentVersionForIP(ip string) (string, error)

	// GetCommunicationHosts returns, on success, the list of communication hosts used for available
	// communication endpoints that the Dynatrace OneAgent can use to connect to.
	//
	// Returns an error if there was also an error response from the server.
	GetConnectionInfo() (ConnectionInfo, error)

	// GetCommunicationHostForClient returns a CommunicationHost for the client's API URL. Or error, if failed to be parsed.
	GetCommunicationHostForClient() (CommunicationHost, error)

	// SendEvent posts events to dynatrace API
	SendEvent(eventData *EventData) error

	// GetEntityIDForIP returns the entity id for a given IP address.
	//
	// Returns an error in case the lookup failed.
	GetEntityIDForIP(ip string) (string, error)

	// GetTokenScopes returns the list of scopes assigned to a token if successful.
	GetTokenScopes(token string) (TokenScopes, error)
}

// Known OS values.
const (
	OsWindows = "windows"
	OsUnix    = "unix"
	OsAix     = "aix"
	OsSolaris = "solaris"
)

// Known installer types.
const (
	InstallerTypeDefault    = "default"
	InstallerTypeUnattended = "default-unattended"
	InstallerTypePaasZip    = "paas"
	InstallerTypePaasSh     = "paas-sh"
)

// Known token scopes
const (
	TokenScopeInstallerDownload = "InstallerDownload"
	TokenScopeDataExport        = "DataExport"
)

// NewClient creates a REST client for the given API base URL and authentication tokens.
// Returns an error if a token or the URL is empty.
//
// The API base URL is different for managed and SaaS environments:
//  - SaaS: https://{environment-id}.live.dynatrace.com/api
//  - Managed: https://{domain}/e/{environment-id}/api
//
// opts can be used to customize the created client, entries must not be nil.
func NewClient(url, apiToken, paasToken string, opts ...Option) (Client, error) {
	if len(url) == 0 {
		return nil, errors.New("url is empty")
	}
	if len(apiToken) == 0 || len(paasToken) == 0 {
		return nil, errors.New("token is empty")
	}

	if strings.HasSuffix(url, "/") {
		url = url[:len(url)-1]
	}

	dc := &dynatraceClient{
		url:       url,
		apiToken:  apiToken,
		paasToken: paasToken,
		logger:    log.Log.WithName("dynatrace.client"),

		hostCache: make(map[string]hostInfo),
		httpClient: &http.Client{
			Transport: http.DefaultTransport.(*http.Transport).Clone(),
		},
	}

	for _, opt := range opts {
		opt(dc)
	}
	return dc, nil
}

// Option can be passed to NewClient and customizes the created client instance.
type Option func(*dynatraceClient)

// SkipCertificateValidation creates an Option that specifies whether validation of the server's TLS
// certificate should be skipped. The default is false.
func SkipCertificateValidation(skip bool) Option {
	return func(c *dynatraceClient) {
		if skip {
			t := c.httpClient.Transport.(*http.Transport)
			if t.TLSClientConfig == nil {
				t.TLSClientConfig = &tls.Config{}
			}
			t.TLSClientConfig.InsecureSkipVerify = true
		}
	}
}

func Proxy(proxyURL string) Option {
	return func(c *dynatraceClient) {
		p, err := url.Parse(proxyURL)
		if err != nil {
			c.logger.Info("Could not parse proxy URL!")
			return
		}
		t := c.httpClient.Transport.(*http.Transport)
		t.Proxy = http.ProxyURL(p)
	}
}

func Certs(certs []byte) Option {
	return func(c *dynatraceClient) {
		rootCAs := x509.NewCertPool()
		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			c.logger.Info("Failed to append custom certs!")
		}

		t := c.httpClient.Transport.(*http.Transport)
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		}
		t.TLSClientConfig.RootCAs = rootCAs
	}
}

func NetworkZone(networkZone string) Option {
	return func(c *dynatraceClient) {
		c.networkZone = networkZone
	}
}
