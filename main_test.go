package main

import (
	"database/sql"
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

func TestServer(t *testing.T) {
	db, _ := sql.Open("postgres", "user=mike host=localhost dbname=endpoints_test sslmode=disable")

	// Test setup
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	db.Exec("BEGIN")
	defer db.Exec("ROLLBACK")

	const watchpointURL = "test"
	db.Exec("INSERT INTO endpoints (watchpointURL) VALUES ($1)", watchpointURL)

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
		t.Error(err)
	case <-time.NewTimer(10 * time.Second).C:
		t.Fatal("Timed out waiting for server to start")
	case <-rdy:

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
		common.WriteWithSize(conn, data)

		buf, _, err := common.ReadWithSize(conn)
		if err != nil {
			t.Fatal(err)
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
