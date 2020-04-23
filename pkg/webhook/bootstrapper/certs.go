package bootstrapper

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os/exec"
	"path"
)

var serialNumberLimit = new(big.Int).Lsh(big.NewInt(1), 128)

type certs struct {
	rootPublicCertPEM []byte
}

func (cs *certs) generateRootCerts(dir, domain string) error {
	var err error

	keyPath := path.Join(dir, "ca.key")
	certPath := path.Join(dir, "ca.crt")

	if out, err := exec.Command(
		"openssl",
		"genrsa",
		"-out", keyPath,
		"4096").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate root key: %v: %s", err, out)
	}

	if out, err := exec.Command(
		"openssl",
		"req",
		"-x509",
		"-new",
		"-nodes",
		"-key", keyPath,
		"-sha256",
		"-days", "365",
		"-out", certPath,
		"-subj", "/C=AT/ST=UA/L=Linz/O=Dynatrace/OU=CloudPlatform/CN="+domain).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate root certificate: %v: %s", err, out)
	}

	cs.rootPublicCertPEM, err = ioutil.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read root certificate: %w", err)
	}

	return nil
}

func (cs *certs) generateServerCerts(dir, domain string) error {
	caKeyPath := path.Join(dir, "ca.key")
	caCertPath := path.Join(dir, "ca.crt")
	csrPath := path.Join(dir, "tls.csr")
	keyPath := path.Join(dir, "tls.key")
	certPath := path.Join(dir, "tls.crt")

	if out, err := exec.Command(
		"openssl",
		"genrsa",
		"-out", keyPath,
		"4096").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate server key: %v: %s", err, out)
	}

	if out, err := exec.Command(
		"openssl",
		"req",
		"-new",
		"-nodes",
		"-key", keyPath,
		"-sha256",
		"-out", csrPath,
		"-subj", "/C=AT/ST=UA/L=Linz/O=Dynatrace/OU=CloudPlatform/CN="+domain).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate server certificate signing request: %v: %s", err, out)
	}

	if out, err := exec.Command(
		"openssl",
		"x509",
		"-req",
		"-in", csrPath,
		"-CA", caCertPath,
		"-CAkey", caKeyPath,
		"-CAcreateserial",
		"-out", certPath,
		"-days", "7",
		"-sha256").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate server certificate: %v: %s", err, out)
	}

	return nil
}
