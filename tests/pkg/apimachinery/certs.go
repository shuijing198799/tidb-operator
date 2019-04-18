package apimachinery

import (
	"crypto/x509"
	"io/ioutil"
	"os"
	"github.com/golang/glog"
	"k8s.io/client-go/util/cert"
)

type CertContext struct {
	Cert        []byte
	Key         []byte
	SigningCert []byte
}

// Setup the server cert. For example, user apiservers and admission webhooks
// can use the cert to prove their identify to the kube-apiserver
func SetupServerCert(namespaceName, serviceName string) *CertContext {
	certDir, err := ioutil.TempDir("", "test-e2e-server-cert")
	if err != nil {
		glog.Errorf("Failed to create a temp dir for cert generation %v", err)
	}
	defer os.RemoveAll(certDir)
	signingKey, err := cert.NewPrivateKey()
	if err != nil {
		glog.Errorf("Failed to create CA private key %v", err)
	}
	signingCert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: "e2e-server-cert-ca"}, signingKey)
	if err != nil {
		glog.Errorf("Failed to create CA cert for apiserver %v", err)
	}
	caCertFile, err := ioutil.TempFile(certDir, "ca.crt")
	if err != nil {
		glog.Errorf("Failed to create a temp file for ca cert generation %v", err)
	}
	if err := ioutil.WriteFile(caCertFile.Name(), cert.EncodeCertPEM(signingCert), 0644); err != nil {
		glog.Errorf("Failed to write CA cert %v", err)
	}
	key, err := cert.NewPrivateKey()
	if err != nil {
		glog.Errorf("Failed to create private key for %v", err)
	}
	signedCert, err := cert.NewSignedCert(
		cert.Config{
			CommonName: serviceName + "." + namespaceName + ".svc",
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
		key, signingCert, signingKey,
	)
	if err != nil {
		glog.Errorf("Failed to create cert%v", err)
	}
	certFile, err := ioutil.TempFile(certDir, "server.crt")
	if err != nil {
		glog.Errorf("Failed to create a temp file for cert generation %v", err)
	}
	keyFile, err := ioutil.TempFile(certDir, "server.key")
	if err != nil {
		glog.Errorf("Failed to create a temp file for key generation %v", err)
	}
	if err = ioutil.WriteFile(certFile.Name(), cert.EncodeCertPEM(signedCert), 0600); err != nil {
		glog.Errorf("Failed to write cert file %v", err)
	}
	if err = ioutil.WriteFile(keyFile.Name(), cert.EncodePrivateKeyPEM(key), 0644); err != nil {
		glog.Errorf("Failed to write key file %v", err)
	}
	return &CertContext{
		Cert:        cert.EncodeCertPEM(signedCert),
		Key:         cert.EncodePrivateKeyPEM(key),
		SigningCert: cert.EncodeCertPEM(signingCert),
	}
}
