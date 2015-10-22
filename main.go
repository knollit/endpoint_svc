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

	"github.com/mikeraimondi/knollit/common"
)

var (
	dbAddr   = flag.String("db-addr", os.Getenv("POSTGRES_PORT_5432_TCP_ADDR"), "Database address")
	dbPW     = flag.String("db-pw", os.Getenv("POSTGRES_PASSWORD"), "Database password")
	caPath   = flag.String("ca-path", os.Getenv("TLS_CA_PATH"), "Path to CA file")
	certPath = flag.String("cert-path", os.Getenv("TLS_CERT_PATH"), "Path to cert file")
	keyPath  = flag.String("key-path", os.Getenv("TLS_KEY_PATH"), "Path to private key file")
)

func main() {
	connStr := fmt.Sprintf("user=postgres host=%v password=%v dbname=postgres sslmode=disable", *dbAddr, *dbPW)
	db, _ := sql.Open("postgres", connStr)

	// Load server cert
	cert, err := tls.LoadX509KeyPair(*certPath, *keyPath)
	if err != nil {
		log.Fatal("Failed to open server cert and/or key: ", err)
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(*caPath)
	if err != nil {
		log.Fatal("Failed to open CA cert: ", err)
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		log.Fatal("Failed to parse CA cert")
	}

	server := &server{
		DB: db,
		TLSConf: &tls.Config{
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
		},
	}
	defer func() {
		if err := server.Close(); err != nil {
			log.Println("Failed to close server: ", err)
		}
	}()

	log.Fatal(server.run(":13800"))
}

type server struct {
	DB       *sql.DB
	TLSConf  *tls.Config
	listener net.Listener
	ready    chan int
}

func (s *server) handler(conn net.Conn) {
	defer conn.Close()
	buf, _, err := common.ReadWithSize(conn)
	log.Println(buf)
	if err != nil {
		log.Print(err)
		// TODO send error
		return
	}
}

func (s *server) run(addr string) error {
	if err := s.DB.Ping(); err != nil {
		return err
	}
	l, err := tls.Listen("tcp", addr, s.TLSConf)
	if err != nil {
		return err
	}
	s.listener = l

	log.Printf("Listening for requests on %s...\n", addr)
	if s.ready != nil {
		s.ready <- 1
	}
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}
		go s.handler(conn)
	}
}

func (s *server) Close() error {
	if err := s.listener.Close(); err != nil {
		log.Println("Failed to close TCP listener cleanly: ", err)
	}
	if err := s.DB.Close(); err != nil {
		log.Println("Failed to close database connection cleanly: ", err)
	}

	return nil
}
