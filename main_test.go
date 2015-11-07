package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/mikeraimondi/knollit/common"
	endpointProto "github.com/mikeraimondi/knollit/endpoints/proto"
)

type logWriter struct {
	*testing.T
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	l.Log(string(p))
	return len(p), nil
}

func TestEndpointIndexWithOne(t *testing.T) {
	// Create test database. Ignore errors.
	db, _ := sql.Open("postgres", "user=mike host=localhost sslmode=disable")
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	db.Exec("CREATE DATABASE endpoints_test")
	db.Close()

	// Setup test database
	// TODO don't use mike
	db, _ = sql.Open("postgres", "user=mike host=localhost dbname=endpoints_test sslmode=disable")
	setupSQL, _ := ioutil.ReadFile("db/db.sql")
	if _, err := db.Exec(string(setupSQL)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("BEGIN"); err != nil {
		t.Fatal(err)
	}
	defer db.Exec("ROLLBACK")

	// Test-specific setup
	const watchpointURL = "test"
	if _, err := db.Exec("INSERT INTO endpoints (watchpointURL) VALUES ($1)", watchpointURL); err != nil {
		t.Fatal(err)
	}

	// Server setup
	const addr = ":13900"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	rdy := make(chan int)
	server := &server{
		db:       db,
		listener: l,
		ready:    rdy,
		logger:   log.New(&logWriter{t}, "", log.Lmicroseconds),
	}

	// Server startup
	errs := make(chan error)
	go func() {
		errs <- server.run()
	}()
	select {
	case err = <-errs:
		t.Fatal(err)
	case <-time.NewTimer(10 * time.Second).C:
		t.Fatal("Timed out waiting for server to start")
	case <-rdy:
		defer server.Close()

		// Begin test
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		data, err := proto.Marshal(&endpointProto.Request{
			Action: endpointProto.Request_INDEX,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := common.WriteWithSize(conn, data); err != nil {
			t.Fatal(err)
		}

		buf, _, err := common.ReadWithSize(conn)
		if err != nil {
			t.Fatalf("Error reading response from server: %v", err)
		}
		endpointMsg := &endpointProto.Endpoint{}
		if err := proto.Unmarshal(buf, endpointMsg); err != nil {
			t.Fatal(err)
		}

		if endpointMsg.WatchpointURL != watchpointURL {
			t.Fatalf("Expected %v for watchpointURL, got %v", watchpointURL, endpointMsg.WatchpointURL)
		}
	}
}
