package registry

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/tinyzimmer/k3p/pkg/log"
)

// These should be moved to types (or eventual apis) package
const (
	RegistryUser = "registry"

	RegistryNamespace  = "kube-system"
	RegistryTLSSecret  = "registry-tls"
	RegistryAuthSecret = "registry-htpasswd"
	KubenabTLSSecret   = "kubenab-tls"

	RegistryK8sAppName = "private-registry"
	KubenabK8sAppName  = "kubenab"

	RegistryCAPath = "/etc/rancher/k3s/registry-ca.crt"

	KubenabImage = "docker.bintray.io/kubenab:0.3.4"
)

// GenerateRegistryAuthSecret will create a kubernetes secret cotaining an htpasswd file
// for registry basic auth.
func GenerateRegistryAuthSecret(secret string) ([]byte, error) {
	// Generate htpasswd file for the registry
	log.Info("Generating secrets for registry authentication")
	passwordBytes, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	htpasswd := append([]byte(RegistryUser), []byte(":")...)
	htpasswd = append(htpasswd, passwordBytes...)
	htpasswd = append(htpasswd, []byte("\n")...)

	return executeTemplate(registryAuthSecretTmpl, map[string]interface{}{
		"RegistryAuthSecret":   RegistryAuthSecret,
		"RegistryNamespace":    RegistryNamespace,
		"RegistryK8sAppName":   RegistryK8sAppName,
		"RegistryAuthHtpasswd": string(htpasswd),
	})
}

// GenerateRegistryDeployments will generate Deployments objects for the registry.
func GenerateRegistryDeployments(dataImageName string) ([]byte, error) {
	return executeTemplate(registryDeploymentsTmpl, map[string]interface{}{
		"KubenabImage":       KubenabImage,
		"KubenabTLSSecret":   KubenabTLSSecret,
		"KubenabK8sAppName":  KubenabK8sAppName,
		"RegistryK8sAppName": RegistryK8sAppName,
		"RegistryNamespace":  RegistryNamespace,
		"RegistryAuthSecret": RegistryAuthSecret,
		"RegistryTLSSecret":  RegistryTLSSecret,
		"RegistryDataImage":  dataImageName,
	})
}

// GenerateRegistryServices will generate Service objects for the registry.
func GenerateRegistryServices(port int) ([]byte, error) {
	return executeTemplate(registryServicesTmpl, map[string]interface{}{
		"KubenabK8sAppName":  KubenabK8sAppName,
		"RegistryK8sAppName": RegistryK8sAppName,
		"RegistryNamespace":  RegistryNamespace,
		"RegistryNodePort":   strconv.Itoa(port),
	})
}

// GenerateRegistriesYaml will generate the registries.yaml used to configure containerd.
func GenerateRegistriesYaml(secret string, port int) ([]byte, error) {
	return executeTemplate(registriesYamlTmpl, map[string]interface{}{
		"RegistryNodePort": strconv.Itoa(port),
		"Username":         RegistryUser,
		"Password":         secret,
		"RegistryCAPath":   RegistryCAPath,
	})
}

// GenerateRegistryTLSSecrets will generate secrets and configurations for registry TLS.
func GenerateRegistryTLSSecrets(name string) (caCertBytes, k8sManifests []byte, err error) {
	// Generate certificates for the registry
	// TODO: Allow user to supply certificates
	caCert, caPriv, err := generateCACertificate(name)
	if err != nil {
		return nil, nil, err
	}
	registryCert, registryPriv, err := generateRegistryCertificate(caCert, caPriv, name)
	if err != nil {
		return nil, nil, err
	}
	caCertPEM, _, err := encodeToPEM(caCert, caPriv)
	if err != nil {
		return nil, nil, err
	}
	registryCertPEM, registryKeyPEM, err := encodeToPEM(registryCert, registryPriv)
	if err != nil {
		return nil, nil, err
	}
	manifests, err := executeTemplate(registryTLSTmpl, map[string]interface{}{
		"KubenabTLSSecret":   KubenabTLSSecret,
		"KubenabK8sAppName":  KubenabK8sAppName,
		"RegistryTLSSecret":  RegistryTLSSecret,
		"RegistryK8sAppName": RegistryK8sAppName,
		"RegistryNamespace":  RegistryNamespace,
		"TLSCertificate":     string(registryCertPEM),
		"TLSPrivateKey":      string(registryKeyPEM),
		"TLSCACertificate":   string(caCertPEM),
	})
	return caCertPEM, manifests, err
}

func generateCACertificate(name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Generate a 4096-bit RSA private key
	caPriv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	fixName := strings.Replace(name, "_", "-", -1)
	caCert := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   fmt.Sprintf("%s-registry-ca", fixName),
			Organization: []string{fmt.Sprintf("%s-private-registry", fixName)},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 10), // 10 years - obviously needs to be handled better
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDerBytes, err := x509.CreateCertificate(rand.Reader, caCert, caCert, caPriv.Public(), caPriv)
	if err != nil {
		return nil, nil, err
	}
	caCertSigned, err := x509.ParseCertificate(caDerBytes)
	if err != nil {
		return nil, nil, err
	}
	return caCertSigned, caPriv, nil
}

func generateRegistryCertificate(caCert *x509.Certificate, caKey *rsa.PrivateKey, name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	fixName := strings.Replace(name, "_", "-", -1)
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   fmt.Sprintf("%s-private-registry", fixName),
			Organization: []string{fmt.Sprintf("%s-private-registry", fixName)},
		},
		DNSNames:              []string{"localhost", "kubenab.kube-system.svc"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 10), // 10 years - obviously needs to be handled better
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, cert, caCert, priv.Public(), caKey)
	if err != nil {
		return nil, nil, err
	}
	certSigned, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, nil, err
	}
	return certSigned, priv, nil
}

func encodeToPEM(rawCert *x509.Certificate, rawKey *rsa.PrivateKey) (cert, key []byte, err error) {
	var certout bytes.Buffer

	// encode the certificate
	if err := pem.Encode(&certout, &pem.Block{Type: "CERTIFICATE", Bytes: rawCert.Raw}); err != nil {
		return nil, nil, err
	}
	certBytes := certout.Bytes()

	var keyout bytes.Buffer

	// encode the private key
	if err := pem.Encode(&keyout, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rawKey)}); err != nil {
		return nil, nil, err
	}
	keyBytes := keyout.Bytes()

	return certBytes, keyBytes, nil
}
