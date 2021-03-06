package main

import (
	"errors"
	"flag"
	"net"
	"os"
	"testing"

	"github.com/google/flatbuffers/go"
	"github.com/knollit/coelacanth"
	ct "github.com/knollit/coelacanth/testing"
	"github.com/knollit/endpoint_svc/endpoints"
	"github.com/mikeraimondi/prefixedio"
)

func TestMain(m *testing.M) {
	flag.Parse()
	exitCode := m.Run()
	ct.RunAfterCallbacks()
	os.Exit(exitCode)
}

func TestEndpointIndexWithOne(t *testing.T) {
	ct.RunWithServer(t, handler, func(s *coelacanth.Server, addr string) {
		// Test-specific setup
		const URL = "test"
		const org = "5ff0fcbd-8b51-11e5-a171-df11d9bd7d62"
		const schema = "some fake schema"
		if _, err := s.DB.Exec("INSERT INTO endpoints (URL, organization_id, schema) VALUES ($1, $2, $3)", URL, org, schema); err != nil {
			t.Fatal(err)
		}

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
		if _, err := prefixedio.WriteBytes(conn, endpointReq.toFlatBufferBytes(b)); err != nil {
			t.Fatal(err)
		}

		var buf prefixedio.Buffer
		_, err = buf.ReadFrom(conn)
		if err != nil {
			t.Fatalf("Error reading response from server: %v", err)
		}
		endpointMsg := endpoints.GetRootAsEndpoint(buf.Bytes(), 0)

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
	ct.RunWithServer(t, handler, func(s *coelacanth.Server, addr string) {
		// Test-specific setup
		const id2 = "5ff0fcbd-8b51-11e5-a171-df11d9bd7d62"
		const URL2 = "test2"
		const org2 = "5ff0fcbd-8b51-11e5-a171-df11d9bd7d63"
		if _, err := s.DB.Exec("INSERT INTO endpoints (id, url, organization_id) VALUES ($1, $2, $3)", id2, URL2, org2); err != nil {
			t.Fatal(err)
		}
		const id = "5ff0fcbc-8b51-11e5-a171-df11d9bd7d64"
		const URL = "test"
		const org = "5ff0fcbd-8b51-11e5-a171-df11d9bd7d65"
		if _, err := s.DB.Exec("INSERT INTO endpoints (id, url, organization_id) VALUES ($1, $2, $3)", id, URL, org); err != nil {
			t.Fatal(err)
		}

		// Begin test
		conn, err := net.Dial("tcp", addr)
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

		if _, err := prefixedio.WriteBytes(conn, endpointReq.toFlatBufferBytes(b)); err != nil {
			t.Fatal(err)
		}

		var buf prefixedio.Buffer
		_, err = buf.ReadFrom(conn)
		if err != nil {
			t.Fatalf("Error reading response from server: %v", err)
		}
		endpointMsg := endpoints.GetRootAsEndpoint(buf.Bytes(), 0)

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

func TestEndpointReadNotFound(t *testing.T) {
	ct.RunWithServer(t, handler, func(s *coelacanth.Server, addr string) {
		// Test-specific setup
		const id = "5ff0fcbc-8b51-11e5-a171-df11d9bd7d64"

		// Begin test
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		b := flatbuffers.NewBuilder(0)
		endpointReq := endpoint{
			ID:     id,
			Action: endpoints.ActionRead,
		}

		if _, err := prefixedio.WriteBytes(conn, endpointReq.toFlatBufferBytes(b)); err != nil {
			t.Fatal(err)
		}

		var buf prefixedio.Buffer
		_, err = buf.ReadFrom(conn)
		if err != nil {
			t.Fatalf("error reading response from server: %v", err)
		}
		endpointMsg := endpoints.GetRootAsEndpoint(buf.Bytes(), 0)

		if msgID := string(endpointMsg.Id()); msgID != id {
			t.Fatalf("ID does not match. expected: %v. actual: %v.\n", id, msgID)
		}
		if errorMsg := string(endpointMsg.Error()); errorMsg != notFoundErrMsg {
			t.Fatalf("error message does not match. expected: %v. actual: %v.\n", notFoundErrMsg, errorMsg)
		}
	})
}

func TestAllEndpoints(t *testing.T) {
	ct.RunWithDB(t, func(db *ct.TestDB) {
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
		err:            errors.New("some error"),
	}
	buf := e.toFlatBufferBytes(flatbuffers.NewBuilder(0))
	eMsg := endpoints.GetRootAsEndpoint(buf, 0)
	if id := string(eMsg.Id()); e.ID != id {
		t.Fatalf("ID does not match. expected: %v. actual: %v.\n", e.ID, id)
	}
	if org := string(eMsg.OrganizationID()); e.OrganizationID != org {
		t.Fatalf("organization ID does not match. expected: %v. actual: %v.\n", e.OrganizationID, org)
	}
	if url := string(eMsg.URL()); e.URL != url {
		t.Fatalf("URL does not match. expected: %v. actual: %v.\n", e.URL, url)
	}
	if e.Action != eMsg.Action() {
		t.Fatalf("action does not match. expected: %v. actual: %v.\n", e.Action, eMsg.Action)
	}
	if e.err.Error() != string(eMsg.Error()) {
		t.Fatalf("error message does not match. expected: %v. actual: %s.\n", e.err, eMsg.Error())
	}
}

func TestEndpointByIDNoResults(t *testing.T) {
	ct.RunWithDB(t, func(db *ct.TestDB) {
		res, err := endpointByID(db, "5ff0fcbc-8b51-11e5-a171-df11d9bd7d64")
		if err != nil {
			t.Fatal(err)
		}
		if res.err.Error() != notFoundErrMsg {
			t.Fatalf("error message does not match. expected: %v. actual: %v.\n", notFoundErrMsg, res.err)
		}
	})
}

func TestEndpointNew(t *testing.T) {
	ct.RunWithServer(t, handler, func(s *coelacanth.Server, addr string) {
		// Test-specific setup
		const URL = "test"
		const org = "5ff0fcbd-8b51-11e5-a171-df11d9bd7d62"
		const schema = "some fake schema"

		// Begin test
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		b := flatbuffers.NewBuilder(0)
		endpointReq := endpoint{
			OrganizationID: org,
			URL:            URL,
			Action:         endpoints.ActionNew,
		}
		if _, err := prefixedio.WriteBytes(conn, endpointReq.toFlatBufferBytes(b)); err != nil {
			t.Fatal(err)
		}

		var buf prefixedio.Buffer
		_, err = buf.ReadFrom(conn)
		if err != nil {
			t.Fatalf("error reading response from server: %v\n", err)
		}
		endpointMsg := endpoints.GetRootAsEndpoint(buf.Bytes(), 0)

		id := string(endpointMsg.Id())
		if len(id) <= 24 {
			t.Fatalf("UUID does not match. expected: a 24 char ID. actual: %v\n", id)
		}
		if string(endpointMsg.URL()) != URL {
			t.Fatalf("URL does not match. expected: %v. actual %v\n", URL, endpointMsg.URL)
		}
		if msgOrgID := string(endpointMsg.OrganizationID()); msgOrgID != org {
			t.Fatalf("organization does not match. expected: %v. actual: %v\n", org, msgOrgID)
		}
		if newEndpoint, err := endpointByID(s.DB, id); err != nil {
			t.Fatal("error getting new endpoint from DB: ", err)
		} else if newEndpoint.URL != URL {
			t.Fatalf("URL does not match in DB. expected: %v. actual %v\n", URL, newEndpoint.URL)
		}
	})
}
