package main

import (
	"database/sql"

	"github.com/google/flatbuffers/go"
	"github.com/mikeraimondi/knollit/endpoints/endpoints"
)

type endpoint struct {
	ID             string
	OrganizationID string
	URL            string
	Action         int8
	err            error
}

func allEndpoints(db *sql.DB) (endpoints []endpoint, err error) {
	rows, err := db.Query("SELECT id, organization_id, URL FROM endpoints")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var org string
		var url string
		if err = rows.Scan(&id, &org, &url); err != nil {
			return
		}
		endpoint := endpoint{
			ID:             id,
			OrganizationID: org,
			URL:            url,
		}
		endpoints = append(endpoints, endpoint)
	}
	return
}

func endpointByID(db *sql.DB, id string) (e *endpoint, err error) {
	row := db.QueryRow("SELECT id, organization_id, URL FROM endpoints WHERE id = $1 LIMIT 1", id)
	var org string
	var url string
	if err = row.Scan(&id, &org, &url); err != nil {
		return
	}
	e = &endpoint{
		ID:             id,
		OrganizationID: org,
		URL:            url,
	}
	return
}

func (e *endpoint) toFlatBufferBytes(b *flatbuffers.Builder) []byte {
	b.Reset()

	idPosition := b.CreateByteString([]byte(e.ID))
	orgPosition := b.CreateByteString([]byte(e.OrganizationID))
	urlPosition := b.CreateByteString([]byte(e.URL))

	endpoints.EndpointStart(b)

	endpoints.EndpointAddId(b, idPosition)
	endpoints.EndpointAddOrganizationID(b, orgPosition)
	endpoints.EndpointAddURL(b, urlPosition)
	endpoints.EndpointAddAction(b, e.Action)

	endpointPosition := endpoints.EndpointEnd(b)
	b.Finish(endpointPosition)
	return b.Bytes[b.Head():]
}
