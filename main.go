package main

import (
	"net/http"
	"os"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/building-microservices-with-go/chapter10-services-search/data"
	"github.com/building-microservices-with-go/chapter10-services-search/handlers"
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

	statsdClient, err := statsd.New("127.0.0.1:8125")
	if err != nil {
		log.Fatal(err)
	}
	// prefix every metric with the app name
	statsdClient.Namespace = "chapter10.search."

	search := handlers.NewSearch(store, statsdClient)
	health := handlers.NewHealth(statsdClient)

	http.DefaultServeMux.HandleFunc("/", search.Handle)
	http.DefaultServeMux.HandleFunc("/health", health.Handle)

	logger.WithField("service", "search").Infof("Starting server, listening on %s", address)
	log.WithField("service", "search").Fatal(http.ListenAndServe(address, http.DefaultServeMux))
}
