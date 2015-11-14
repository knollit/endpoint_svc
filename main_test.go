package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	"net"
	"testing"
	"time"

	"github.com/google/flatbuffers/go"
	"github.com/mikeraimondi/knollit/common"
	"github.com/mikeraimondi/knollit/endpoints/endpoints"
)

type logWriter struct {
	*testing.T
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	l.Log(string(p))
	return len(p), nil
}

var dbCreated bool

func setupDB() (db *sql.DB, err error) {
	if !dbCreated {
		// Create test database. Ignore errors.
		db, _ = sql.Open("postgres", "user=mike host=localhost sslmode=disable")
		if err = db.Ping(); err != nil {
			return
		}
		db.Exec("DROP DATABASE IF EXISTS endpoints_test")
		db.Exec("CREATE DATABASE endpoints_test")
		db.Close()
		dbCreated = true
	}

	// Setup test database
	// TODO don't use mike
	db, _ = sql.Open("postgres", "user=mike host=localhost dbname=endpoints_test sslmode=disable")
	setupSQL, _ := ioutil.ReadFile("db/db.sql")
	if _, err = db.Exec(string(setupSQL)); err != nil {
		return
	}
	_, err = db.Exec("BEGIN")
	return
}

func TestEndpointIndexWithOne(t *testing.T) {
	db, err := setupDB()
	if err != nil {
		t.Fatal("Error setting up DB: ", err)
	}
	defer db.Exec("ROLLBACK")

	// Test-specific setup
	const watchpointURL = "test"
	const org = "testOrg"
	if _, err := db.Exec("INSERT INTO endpoints (watchpointURL, organization) VALUES ($1, $2)", watchpointURL, org); err != nil {
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

		b := flatbuffers.NewBuilder(0)
		endpointReq := endpoint{
			Action: endpoints.ActionIndex,
		}
		if _, err := common.WriteWithSize(conn, endpointReq.toFlatBufferBytes(b)); err != nil {
			t.Fatal(err)
		}

		buf, _, err := common.ReadWithSize(conn)
		if err != nil {
			t.Fatalf("Error reading response from server: %v", err)
		}
		endpointMsg := endpoints.GetRootAsEndpoint(buf, 0)

		if len(string(endpointMsg.Id())) <= 24 {
			t.Fatalf("Expected UUID for ID, got %v", string(endpointMsg.Id()))
		}
		if string(endpointMsg.WatchpointURL()) != watchpointURL {
			t.Fatalf("Expected %v for watchpointURL, got %v", watchpointURL, endpointMsg.WatchpointURL)
		}
		if string(endpointMsg.Organization()) != org {
			t.Fatalf("Expected %v for organization, got %v", org, endpointMsg.Organization)
		}
	}
}

func TestAllEndpoints(t *testing.T) {
	db, err := setupDB()
	if err != nil {
		t.Fatal("Error setting up DB: ", err)
	}
	defer db.Exec("ROLLBACK")

	// Test-specific setup
	const watchpointURL = "test"
	const org = "testOrg"
	if _, err := db.Exec("INSERT INTO endpoints (watchpointURL, organization) VALUES ($1, $2)", watchpointURL, org); err != nil {
		t.Fatal(err)
	}

	endpoints, err := allEndpoints(db)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(endpoints); l != 1 {
		t.Fatalf("Expected 1 endpoint, got %v", l)
	}
	if endpoints[0].WatchpointURL != watchpointURL {
		t.Fatalf("Expected %v for URL, got %v", watchpointURL, endpoints[0].WatchpointURL)
	}
	if endpoints[0].Organization != org {
		t.Fatalf("Expected %v for organization, got %v", org, endpoints[0].Organization)
	}
}
