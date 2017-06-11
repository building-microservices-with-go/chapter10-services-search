package main

import (
	"net/http"
	"os"

	"github.com/building-microservices-with-go/chapter11-services-search/data"
	"github.com/building-microservices-with-go/chapter11-services-search/handlers"
	log "github.com/sirupsen/logrus"
)

const address = ":8082"

func main() {
	var logger = &log.Logger{
		Out:       os.Stdout,
		Formatter: new(log.TextFormatter),
		Level:     log.DebugLevel,
	}

	store, err := data.NewMySQLStore(os.Getenv("MYSQL_CONNECTION"))
	if err != nil {
		log.Fatal(err)
	}

	handler := handlers.Search{DataStore: store}
	http.DefaultServeMux.HandleFunc("/", handler.ServeHTTP)

	logger.WithField("service", "search").Infof("Starting server, listening on %s", address)
	log.WithField("service", "search").Fatal(http.ListenAndServe(address, http.DefaultServeMux))
}
