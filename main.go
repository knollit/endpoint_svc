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
	"sync"

	"github.com/google/flatbuffers/go"
	_ "github.com/lib/pq"

	"github.com/mikeraimondi/knollit/common"
	"github.com/mikeraimondi/knollit/endpoints/endpoints"
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
	l, err := tls.Listen("tcp", ":13800", tlsConf)
	if err != nil {
		logger.Fatal(err)
	}
	server := &server{
		db:       db,
		logger:   logger,
		listener: l,
	}
	defer func() {
		if err := server.Close(); err != nil {
			server.logger.Println("Failed to close server: ", err)
		}
	}()

	server.logger.Fatal(server.run())
}

type server struct {
	db          *sql.DB
	logger      *log.Logger
	listener    net.Listener
	ready       chan int
	builderPool *sync.Pool
}

func (s *server) handler(conn net.Conn) {
	defer conn.Close()
	buf, _, err := common.ReadWithSize(conn)
	if err != nil {
		s.logger.Print("Error reading request: ", err)
		// TODO send error
		return
	}
	req := endpoints.GetRootAsEndpoint(buf, 0)
	if req.Action() == endpoints.ActionNew {
		return
	}
	endPoints, err := allEndpoints(s.db)
	if err != nil {
		s.logger.Print(err)
		// TODO send error
		return
	}
	b := s.builderPool.Get().(*flatbuffers.Builder)
	defer s.builderPool.Put(b)
	for _, e := range endPoints {
		if _, err := common.WriteWithSize(conn, e.toFlatBufferBytes(b)); err != nil {
			s.logger.Print(err)
		}
	}
	return

}

func (s *server) run() error {
	if err := s.db.Ping(); err != nil {
		return err
	}

	s.builderPool = &sync.Pool{
		New: func() interface{} {
			return flatbuffers.NewBuilder(0)
		},
	}

	s.logger.Printf("Listening for requests on %s...\n", s.listener.Addr())
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
		s.logger.Println("Failed to close TCP listener cleanly: ", err)
	}
	if err := s.db.Close(); err != nil {
		s.logger.Println("Failed to close database connection cleanly: ", err)
	}

	return nil
}
