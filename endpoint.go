package main

import (
	"database/sql"

	"github.com/google/flatbuffers/go"
	"github.com/mikeraimondi/knollit/endpoints/endpoints"
)

type endpoint struct {
	ID            string
	WatchpointURL string
	Action        int8
	err           error
}

func allEndpoints(db *sql.DB) (endpoints []endpoint, err error) {
	rows, err := db.Query("SELECT id, watchpointURL FROM endpoints")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var watchpointURL string
		if err = rows.Scan(&id, &watchpointURL); err != nil {
			return
		}
		endpoint := endpoint{
			ID:            id,
			WatchpointURL: watchpointURL,
		}
		endpoints = append(endpoints, endpoint)
	}
	return
}

func (e *endpoint) toFlatBufferBytes(b *flatbuffers.Builder) []byte {
	b.Reset()

	idPosition := b.CreateByteString([]byte(e.ID))
	urlPosition := b.CreateByteString([]byte(e.WatchpointURL))

	endpoints.EndpointStart(b)

	endpoints.EndpointAddId(b, idPosition)
	endpoints.EndpointAddWatchpointURL(b, urlPosition)
	endpoints.EndpointAddAction(b, e.Action)

	endpointPosition := endpoints.EndpointEnd(b)
	b.Finish(endpointPosition)
	return b.Bytes[b.Head():]
}
