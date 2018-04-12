package dynatrace_client

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

// Client is the interface for the Dynatrace REST API client.
type Client interface {
	// GetVersionForLatest gets the latest agent version for the given OS and installer type.
	GetVersionForLatest(os, installerType string) (string, error)

	// GetVersionForIp returns the agent version running on the host with the given IP address.
	GetVersionForIp(ip net.IP) (string, error)
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

	c := &client{
		url:       url,
		apiToken:  apiToken,
		paasToken: paasToken,

		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// Option can be passed to NewClient and customizes the created client instance.
type Option func(*client)

// SkipCertificateValidation creates an Option that specifies whether validation of the server's TLS
// certificate should be skipped. The default is false.
func SkipCertificateValidation(skip bool) Option {
	return func(c *client) {
		if skip {
			c.httpClient = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			}
		} else {
			c.httpClient = http.DefaultClient
		}
	}
}

// client implements the Client interface.
type client struct {
	url       string
	apiToken  string
	paasToken string

	httpClient *http.Client
}

// GetVersionForLatest gets the latest agent version for the given OS and installer type.
// Returns the version as received from the server on success.
//
// Returns an error for the following conditions:
//  - os or installerType is empty
//  - IO error or unexpected response
//  - error response from the server (e.g. authentication failure)
//  - the agent version is not set or empty
func (c *client) GetVersionForLatest(os, installerType string) (string, error) {
	if len(os) == 0 || len(installerType) == 0 {
		return "", errors.New("os or installerType is empty")
	}

	resp, err := c.makeRequest("%s/v1/deployment/installer/agent/%s/%s/latest/metainfo?Api-Token=%s",
		c.url, os, installerType, c.paasToken)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return readLatestVersion(resp.Body)
}

// GetVersionForIp returns the agent version running on the host with the given IP address.
// Returns the version string formatted as "Major.Minor.Revision.Timestamp" on success.
//
// Returns an error for the following conditions:
//  - the IP is invalid (nil or empty)
//  - IO error or unexpected response
//  - error response from the server (e.g. authentication failure)
//  - a host with the given IP cannot be found
//  - the agent version for the host is not set
func (c *client) GetVersionForIp(ip net.IP) (string, error) {
	if len(ip) == 0 {
		return "", errors.New("ip is invalid")
	}

	resp, err := c.makeRequest("%s/v1/entity/infrastructure/hosts?Api-Token=%s", c.url, c.apiToken)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return readVersionForIp(resp.Body, ip)
}

// makeRequest does an HTTP request by formatting the URL from the given arguments and returns the response.
// The response body must be closed by the caller when no longer used.
func (c *client) makeRequest(format string, a ...interface{}) (*http.Response, error) {
	url := fmt.Sprintf(format, a...)
	return c.httpClient.Get(url)
}

// serverError represents an error returned from the server (e.g. authentication failure).
type serverError struct {
	Code    float64
	Message string
}

// Error formats the server error code and message.
func (e serverError) Error() string {
	if len(e.Message) == 0 && e.Code == 0 {
		return "unknown server error"
	}
	return fmt.Sprintf("error %d: %s", int64(e.Code), e.Message)
}

// readLatestVersion reads the agent version from the given server response reader.
func readLatestVersion(r io.Reader) (string, error) {
	type jsonResponse struct {
		LatestAgentVersion string

		Error *serverError
	}

	var resp jsonResponse
	switch err := json.NewDecoder(r).Decode(&resp); {
	case err != nil:
		return "", err
	case resp.Error != nil:
		return "", resp.Error
	}

	v := resp.LatestAgentVersion
	if len(v) == 0 {
		return "", errors.New("agent version not set")
	}
	return v, nil
}

// readVersionForIp reads the agent version of the given host from the given server response reader.
func readVersionForIp(r io.Reader, ip net.IP) (string, error) {
	type jsonHost struct {
		IpAddresses  []string
		AgentVersion *struct {
			Major     int
			Minor     int
			Revision  int
			Timestamp string
		}
	}

	buf := bufio.NewReader(r)
	// Server sends an array of hosts or an error object, check which one it is
	switch b, err := buf.Peek(1); {
	case err != nil:
		return "", err

	case b[0] == '{':
		// Try decoding an error response
		var resp struct{ Error *serverError }
		switch err = json.NewDecoder(buf).Decode(&resp); {
		case err != nil:
			return "", err
		case resp.Error != nil:
			return "", resp.Error
		default:
			return "", errors.New("unexpected response from server")
		}

	case b[0] != '[':
		return "", errors.New("unexpected response from server")
	}

	// Try decoding a successful response
	var resp []jsonHost
	if err := json.NewDecoder(buf).Decode(&resp); err != nil {
		return "", err
	}

	ipStr := ip.String()
	for _, host := range resp {
		if containsString(host.IpAddresses, ipStr) {
			v := host.AgentVersion
			if v == nil {
				return "", errors.New("agent version not set for host")
			}
			return fmt.Sprintf("%d.%d.%d.%s", v.Major, v.Minor, v.Revision, v.Timestamp), nil
		}
	}
	return "", errors.New("host not found")
}

// containsString determines whether haystack contains the string needle.
func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
