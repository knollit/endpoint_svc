package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"

	_ "github.com/lib/pq"

	"github.com/knollit/endpoint_svc/endpoints"
	"github.com/mikeraimondi/coelacanth"
	"github.com/mikeraimondi/prefixedio"
)

var (
	dbAddr   = flag.String("db-addr", os.Getenv("POSTGRES_PORT_5432_TCP_ADDR"), "Database address")
	dbPW     = flag.String("db-pw", os.Getenv("POSTGRES_PASSWORD"), "Database password")
	caPath   = flag.String("ca-path", os.Getenv("TLS_CA_PATH"), "Path to CA file")
	certPath = flag.String("cert-path", os.Getenv("TLS_CERT_PATH"), "Path to cert file")
	keyPath  = flag.String("key-path", os.Getenv("TLS_KEY_PATH"), "Path to private key file")
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	connStr := fmt.Sprintf("user=postgres host=%v password=%v dbname=postgres sslmode=disable", *dbAddr, *dbPW)
	db, _ := sql.Open("postgres", connStr)

	// Load server cert
	cert, err := tls.LoadX509KeyPair(*certPath, *keyPath)
	if err != nil {
		logger.Fatal("Failed to open server cert and/or key: ", err)
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(*caPath)
	if err != nil {
		logger.Fatal("Failed to open CA cert: ", err)
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		logger.Fatal("Failed to parse CA cert")
	}

	tlsConf := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		ClientCAs:          caCertPool,
		InsecureSkipVerify: true, //TODO dev only
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}

	serverConf := &coelacanth.Config{
		DB: db,
		ListenerFunc: func(addr string) (net.Listener, error) {
			return tls.Listen("tcp", addr, tlsConf)
		},
		Logger: logger,
	}
	server := coelacanth.NewServer(serverConf)
	defer func() {
		if err := server.Close(); err != nil {
			server.Logger.Println("Failed to close server: ", err)
		}
	}()

	server.Logger.Fatal(server.Run(":13800", handler))
}

func handler(conn net.Conn, s *coelacanth.Server) {
	defer conn.Close()

	buf := s.GetPrefixedBuf()
	defer s.PutPrefixedBuf(buf)
	_, err := buf.ReadFrom(conn)
	if err != nil {
		s.Logger.Print("Error reading request: ", err)
		// TODO send error
		return
	}
	req := endpoints.GetRootAsEndpoint(buf.Bytes(), 0)
	switch req.Action() {
	case endpoints.ActionRead:
		e, err := endpointByID(s.DB, string(req.Id()))
		if err != nil {
			s.Logger.Print(err)
			// TODO send error
			return
		}
		b := s.GetBuilder()
		defer s.PutBuilder(b)
		if _, err := prefixedio.WriteBytes(conn, e.toFlatBufferBytes(b)); err != nil {
			s.Logger.Print(err)
		}
		return
	case endpoints.ActionIndex:
		endPoints, err := allEndpoints(s.DB)
		if err != nil {
			s.Logger.Print(err)
			// TODO send error
			return
		}
		b := s.GetBuilder()
		defer s.PutBuilder(b)
		for _, e := range endPoints {
			if _, err := prefixedio.WriteBytes(conn, e.toFlatBufferBytes(b)); err != nil {
				s.Logger.Print(err)
			}
		}
		return
	}
}
