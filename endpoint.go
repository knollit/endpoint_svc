package main

import (
	"github.com/google/flatbuffers/go"
	"github.com/mikeraimondi/knollit/endpoints/endpoints"
)

type endpoint struct {
	WatchpointURL string
	Action        int8
	err           error
}

func allEndpoints(s *server) (endpoints []endpoint, err error) {
	rows, err := s.db.Query("SELECT watchpointURL FROM endpoints")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var watchpointURL string
		if err = rows.Scan(&watchpointURL); err != nil {
			return
		}
		endpoints = append(endpoints, endpoint{WatchpointURL: watchpointURL})
	}
	return
}

func (e *endpoint) toFlatBufferBytes(b *flatbuffers.Builder) []byte {
	b.Reset()
	urlPosition := b.CreateByteString([]byte(e.WatchpointURL))
	endpoints.EndpointStart(b)
	endpoints.EndpointAddWatchpointURL(b, urlPosition)
	endpoints.EndpointAddAction(b, e.Action)
	endpointPosition := endpoints.EndpointEnd(b)
	b.Finish(endpointPosition)
	return b.Bytes[b.Head():]
}
