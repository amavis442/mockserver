package server

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"
)

func TestGenerateSelfSigned_ReturnsValidTLSCert(t *testing.T) {
	certPEM, keyPEM, err := GenerateSelfSigned()
	if err != nil {
		t.Fatalf("GenerateSelfSigned() failed: %v", err)
	}

	if len(certPEM) == 0 {
		t.Error("certPEM is empty")
	}
	if len(keyPEM) == 0 {
		t.Error("keyPEM is empty")
	}

	// Must load as a valid TLS key pair.
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair() failed: %v", err)
	}

	// Must be a parsable x509 certificate.
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate() failed: %v", err)
	}

	// Must be valid for localhost.
	if err := x509Cert.VerifyHostname("localhost"); err != nil {
		t.Errorf("VerifyHostname(localhost) = %v, want nil", err)
	}

	// Must not be expired.
	if time.Now().After(x509Cert.NotAfter) {
		t.Error("certificate is already expired")
	}
	if time.Now().Before(x509Cert.NotBefore) {
		t.Error("certificate is not yet valid")
	}

	// Must have a reasonable lifetime (at least 1 day).
	if x509Cert.NotAfter.Sub(x509Cert.NotBefore) < 24*time.Hour {
		t.Errorf("certificate lifetime too short: %v", x509Cert.NotAfter.Sub(x509Cert.NotBefore))
	}
}

func TestGenerateSelfSigned_EachCallReturnsDifferentKey(t *testing.T) {
	cert1, key1, err := GenerateSelfSigned()
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	cert2, key2, err := GenerateSelfSigned()
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	// Keys must differ (random generation).
	if string(key1) == string(key2) {
		t.Error("two calls returned identical keys")
	}
	// Certs must differ (different serial numbers).
	if string(cert1) == string(cert2) {
		t.Error("two calls returned identical certificates")
	}
}
