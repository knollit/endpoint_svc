package main

import (
	"database/sql"

	"github.com/google/flatbuffers/go"
	"github.com/mikeraimondi/knollit/endpoints/endpoints"
)

type endpoint struct {
	ID            string
	Organization  string
	WatchpointURL string
	Action        int8
	err           error
}

func allEndpoints(db *sql.DB) (endpoints []endpoint, err error) {
	rows, err := db.Query("SELECT id, organization, watchpointURL FROM endpoints")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var org string
		var watchpointURL string
		if err = rows.Scan(&id, &org, &watchpointURL); err != nil {
			return
		}
		endpoint := endpoint{
			ID:            id,
			Organization:  org,
			WatchpointURL: watchpointURL,
		}
		endpoints = append(endpoints, endpoint)
	}
	return
}

func (e *endpoint) toFlatBufferBytes(b *flatbuffers.Builder) []byte {
	b.Reset()

	idPosition := b.CreateByteString([]byte(e.ID))
	orgPosition := b.CreateByteString([]byte(e.Organization))
	urlPosition := b.CreateByteString([]byte(e.WatchpointURL))

	endpoints.EndpointStart(b)

	endpoints.EndpointAddId(b, idPosition)
	endpoints.EndpointAddOrganization(b, orgPosition)
	endpoints.EndpointAddWatchpointURL(b, urlPosition)
	endpoints.EndpointAddAction(b, e.Action)

	endpointPosition := endpoints.EndpointEnd(b)
	b.Finish(endpointPosition)
	return b.Bytes[b.Head():]
}
