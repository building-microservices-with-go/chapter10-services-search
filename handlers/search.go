package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/building-microservices-with-go/chapter11-services-search/data"
)

type searchRequest struct {
	// Query is the text search query that will be executed by the handler
	Query string `json:"query"`
}

type searchResponse struct {
	Kittens []data.Kitten `json:"kittens"`
}

// Search is an http handler for our microservice
type Search struct {
	dataStore data.Store
	statsd    *statsd.Client
}

func (s *Search) Handle(rw http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	request := &searchRequest{}
	err := decoder.Decode(request)
	if err != nil || len(request.Query) < 1 {
		s.statsd.Incr("search.badrequest", nil, 1)

		log.Println(err)
		http.Error(rw, "Bad Request", http.StatusBadRequest)
		return
	}

	kittens := s.dataStore.Search(request.Query)

	encoder := json.NewEncoder(rw)
	encoder.Encode(searchResponse{Kittens: kittens})

	s.statsd.Incr("search.success", nil, 1)
}

func NewSearch(dataStore data.Store, statsd *statsd.Client) *Search {
	return &Search{
		dataStore: dataStore,
		statsd:    statsd,
	}
}
