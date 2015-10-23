package main

import (
	"bytes"
	"database/sql"
	"io/ioutil"
	"log"
	"net"
	"testing"

	"github.com/mikeraimondi/knollit/common"
)

func TestServer(t *testing.T) {
	var logBuf bytes.Buffer
	defer func() {
		log, err := ioutil.ReadAll(&logBuf)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(string(log))
	}()

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
		logger:   log.New(&logBuf, "", log.LstdFlags),
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
