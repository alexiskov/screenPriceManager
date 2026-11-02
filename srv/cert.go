package srv

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

const (
	CertDir  = "certs"
	CertFile = "cert.pem"
	KeyFile  = "key.pem"
)

// EnsureCertificates проверяет наличие папки и сертификатов.
// Если папки нет — создает. Если сертификатов нет — генерирует самоподписанные.
// Возвращает пути к файлам.
func ensureCertificates(ip, organization, dnsn string) (string, string, error) {
	// 1. Создаем папку certs, если её нет
	if err := os.MkdirAll(CertDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create certs directory: %w", err)
	}

	certPath := filepath.Join(CertDir, CertFile)
	keyPath := filepath.Join(CertDir, KeyFile)

	// 2. Если оба файла уже существуют — используем их
	if _, err := os.Stat(certPath); err == nil {
		if _, err := os.Stat(keyPath); err == nil {

			return certPath, keyPath, nil
		}
	}

	// 3. Генерируем самоподписанный сертификат

	if err := generateSelfSigned(certPath, keyPath, ip, organization, dnsn); err != nil {
		return "", "", err
	}

	return certPath, keyPath, nil
}

func generateSelfSigned(certPath, keyPath, ip, organization, dnsn string) error {
	if certPath == keyPath {
		return fmt.Errorf("certPath and keyPath must differ")
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address: %q", ip)
	}

	if organization == "" {
		organization = "Local Dev Server"
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	dnsNames := []string{}
	if dnsn != "" {
		dnsNames = append(dnsNames, dnsn)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{organization},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    dnsNames,
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), parsedIP},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %w", certPath, err)
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed to write data to %s: %w", certPath, err)
	}

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %w", keyPath, err)
	}
	defer keyOut.Close()
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return fmt.Errorf("unable to marshal private key: %w", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("failed to write data to %s: %w", keyPath, err)
	}

	return nil
}
