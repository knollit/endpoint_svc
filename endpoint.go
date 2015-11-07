package main

type endpoint struct {
	WatchpointURL string
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
