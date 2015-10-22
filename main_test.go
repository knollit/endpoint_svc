package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"io/ioutil"
	"testing"
)

func TestServer(t *testing.T) {
	// TODO setup DB

	db, _ := sql.Open("postgres", "user=mike host=localhost dbname=endpoints_test sslmode=disable")

	cert, err := tls.LoadX509KeyPair("certs/test-server.crt", "certs/test-server.key")
	if err != nil {
		t.Fatal("Failed to open server cert and/or key: ", err)
	}

	caCert, err := ioutil.ReadFile("certs/test-ca.crt")
	if err != nil {
		t.Fatal("Failed to open CA cert: ", err)
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		t.Fatal("Failed to parse CA cert")
	}

	rdy := make(chan int)
	server := &server{
		DB: db,
		TLSConf: &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			ClientAuth:         tls.RequireAndVerifyClientCert,
			ClientCAs:          caCertPool,
			InsecureSkipVerify: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			},
			PreferServerCipherSuites: true,
			MinVersion:               tls.VersionTLS12,
		},
		ready: rdy,
	}

	errs := make(chan error)

	go func() {
		errs <- server.run(":13900")
	}()
	select {
	case err = <-errs:
		t.Error(err)
	case <-rdy:
		// make request here
	}
}
