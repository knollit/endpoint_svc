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
	"github.com/mikeraimondi/knollit/endpoint_svc/endpoints"
)

type logWriter struct {
	*testing.T
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	l.Log(string(p))
	return len(p), nil
}

var dbCreated bool

func runWithDB(t *testing.T, testFunc func(*sql.DB)) {
	if !dbCreated {
		// Create test database. Ignore errors.
		// TODO don't use mike
		db, _ := sql.Open("postgres", "user=mike host=localhost dbname=postgres sslmode=disable")
		if err := db.Ping(); err != nil {
			t.Fatal("Error opening DB: ", err)
		}
		db.Exec("DROP DATABASE IF EXISTS endpoints_test")
		db.Exec("CREATE DATABASE endpoints_test")
		db.Close()
		dbCreated = true
	}

	// Setup test database
	// TODO don't use mike
	db, _ := sql.Open("postgres", "user=mike host=localhost dbname=endpoints_test sslmode=disable")
	setupSQL, _ := ioutil.ReadFile("db/db.sql")
	if _, err := db.Exec(string(setupSQL)); err != nil {
		t.Fatal("Error setting up DB: ", err)
	}
	if _, err := db.Exec("BEGIN"); err != nil {
		t.Fatal("Error starting TX: ", err)
	}
	defer func() {
		if _, err := db.Exec("ROLLBACK"); err != nil {
			t.Fatal("Error rolling back TX: ", err)
		}
	}()
	testFunc(db)
	return
}

func runWithServer(t *testing.T, testFunc func(*server)) {
	runWithDB(t, func(db *sql.DB) {
		// Setup server
		const addr = ":13900" // TODO pick a better number?
		l, err := net.Listen("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		rdy := make(chan int)
		s := &server{
			db:       db,
			listener: l,
			ready:    rdy,
			logger:   log.New(&logWriter{t}, "", log.Lmicroseconds),
		}
		defer s.listener.Close() // TODO better to call s.Close(), but can't because it closes DB

		// Start server on another goroutine
		errs := make(chan error)
		go func() {
			errs <- s.run()
		}()
		select {
		case err := <-errs:
			t.Fatal(err)
		case <-time.NewTimer(10 * time.Second).C:
			t.Fatal("Timed out waiting for server to start")
		case <-rdy:
			testFunc(s)
		}
	})
	return
}

func TestEndpointIndexWithOne(t *testing.T) {
	runWithServer(t, func(s *server) {
		// Test-specific setup
		const URL = "test"
		const org = "5ff0fcbd-8b51-11e5-a171-df11d9bd7d62"
		const schema = "some fake schema"
		if _, err := s.db.Exec("INSERT INTO endpoints (URL, organization_id, schema) VALUES ($1, $2, $3)", URL, org, schema); err != nil {
			t.Fatal(err)
		}

		// Begin test
		conn, err := net.Dial("tcp", s.listener.Addr().String())
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
		if string(endpointMsg.URL()) != URL {
			t.Fatalf("Expected %v for URL, got %v", URL, endpointMsg.URL)
		}
		if msgOrgID := string(endpointMsg.OrganizationID()); msgOrgID != org {
			t.Fatalf("Expected %v for organization, got %v", org, msgOrgID)
		}
		if msgSchema := string(endpointMsg.Schema()); msgSchema != schema {
			t.Fatalf("Expected %v for organization, got %v", schema, msgSchema)
		}
	})
}

func TestEndpointReadWithTwo(t *testing.T) {
	runWithServer(t, func(s *server) {
		// Test-specific setup
		const id2 = "5ff0fcbd-8b51-11e5-a171-df11d9bd7d62"
		const URL2 = "test2"
		const org2 = "5ff0fcbd-8b51-11e5-a171-df11d9bd7d63"
		if _, err := s.db.Exec("INSERT INTO endpoints (id, url, organization_id) VALUES ($1, $2, $3)", id2, URL2, org2); err != nil {
			t.Fatal(err)
		}
		const id = "5ff0fcbc-8b51-11e5-a171-df11d9bd7d64"
		const URL = "test"
		const org = "5ff0fcbd-8b51-11e5-a171-df11d9bd7d65"
		if _, err := s.db.Exec("INSERT INTO endpoints (id, url, organization_id) VALUES ($1, $2, $3)", id, URL, org); err != nil {
			t.Fatal(err)
		}

		// Begin test
		conn, err := net.Dial("tcp", s.listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		b := flatbuffers.NewBuilder(0)
		endpointReq := endpoint{
			ID:             id,
			OrganizationID: org,
			Action:         endpoints.ActionRead,
		}

		if _, err := common.WriteWithSize(conn, endpointReq.toFlatBufferBytes(b)); err != nil {
			t.Fatal(err)
		}

		buf, _, err := common.ReadWithSize(conn)
		if err != nil {
			t.Fatalf("Error reading response from server: %v", err)
		}
		endpointMsg := endpoints.GetRootAsEndpoint(buf, 0)

		if msgID := string(endpointMsg.Id()); msgID != id {
			t.Fatalf("Expected %v for ID, got %v", id, msgID)
		}
		if msgURL := string(endpointMsg.URL()); msgURL != URL {
			t.Fatalf("Expected %v for URL, got %v", URL, msgURL)
		}
		if msgOrg := string(endpointMsg.OrganizationID()); msgOrg != org {
			t.Fatalf("Expected %v for organization, got %v", org, msgOrg)
		}
	})
}

func TestAllEndpoints(t *testing.T) {
	runWithDB(t, func(db *sql.DB) {
		// Test-specific setup
		const URL = "http://test.com"
		const watchpointURL = "test"
		const org = "5ff0fcbc-8b51-11e5-a171-df11d9bd7d64"
		if _, err := db.Exec("INSERT INTO endpoints (URL, organization_id) VALUES ($1, $2)", URL, org); err != nil {
			t.Fatal(err)
		}

		endpoints, err := allEndpoints(db)
		if err != nil {
			t.Fatal(err)
		}
		if l := len(endpoints); l != 1 {
			t.Fatalf("Expected 1 endpoint, got %v", l)
		}
		e := endpoints[0]
		if len(e.ID) <= 24 {
			t.Fatalf("Expected UUID for ID, got %v", e.ID)
		}
		if e.URL != URL {
			t.Fatalf("Expected %v for URL, got %v", URL, e.URL)
		}
		if e.OrganizationID != org {
			t.Fatalf("Expected %v for organization, got %v", org, e.OrganizationID)
		}
	})
}

func TestToFlatBufferBytes(t *testing.T) {
	t.Parallel()
	e := endpoint{
		ID:             "5ff0fcbc-8b51-11e5-a171-df11d9bd7d62",
		OrganizationID: "5ff0fcbc-8b51-11e5-a171-df11d9bd7d63",
		URL:            "http://foo.bar",
		Action:         endpoints.ActionNew,
	}
	buf := e.toFlatBufferBytes(flatbuffers.NewBuilder(0))
	eMsg := endpoints.GetRootAsEndpoint(buf, 0)
	if id := string(eMsg.Id()); e.ID != id {
		t.Fatalf("Expected %v for ID, got %v", e.ID, id)
	}
	if org := string(eMsg.OrganizationID()); e.OrganizationID != org {
		t.Fatalf("Expected %v for Organization, got %v", e.OrganizationID, org)
	}
	if url := string(eMsg.URL()); e.URL != url {
		t.Fatalf("Expected %v for URL, got %v", e.URL, url)
	}
	if e.Action != eMsg.Action() {
		t.Fatalf("Expected %v for Action, got %v", e.Action, eMsg.Action)
	}
}
