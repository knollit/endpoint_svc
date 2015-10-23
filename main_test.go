package main

import (
	"database/sql"
	"log"
	"net"
	"testing"

	"github.com/mikeraimondi/knollit/common"
)

type logWriter struct {
	*testing.T
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	l.Log(string(p))
	return len(p), nil
}

func TestServer(t *testing.T) {
	// TODO setup DB

	db, _ := sql.Open("postgres", "user=mike host=localhost dbname=endpoints_test sslmode=disable")

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
		logger:   log.New(&logWriter{t}, "", log.LstdFlags),
	}

	errs := make(chan error)

	go func() {
		errs <- server.run()
	}()
	select {
	case err = <-errs:
		t.Error(err)
	case <-rdy:
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		common.WriteWithSize(conn, []byte("foo"))
	}
}
