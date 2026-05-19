package main

import (
	"crypto/tls"
	"fmt"

	"github.com/emiago/diago/media"
)

func loadDTLSConfig(certPath, keyPath string) (media.DTLSConfig, error) {
	if certPath == "" || keyPath == "" {
		return media.DTLSConfig{}, fmt.Errorf("dtls cert and key paths are required for DTLS-SRTP (set --dtls_cert/--dtls_key or DTLS_CERT_FILE/DTLS_KEY_FILE)")
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return media.DTLSConfig{}, fmt.Errorf("load dtls certificate: %w", err)
	}

	return media.DTLSConfig{
		Certificates:     []tls.Certificate{cert},
		ServerClientAuth: media.ServerClientAuthNoCert,
	}, nil
}
