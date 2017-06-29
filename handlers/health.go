package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/DataDog/datadog-go/statsd"
)

type Health struct {
	statsd *statsd.Client
}

func (h *Health) Handle(rw http.ResponseWriter, r *http.Request) {
	defer func(startTime time.Time) {
		h.statsd.Timing("health.timing", time.Now().Sub(startTime), nil, 1)
	}(time.Now())

	h.statsd.Incr("health.success", nil, 1)
	fmt.Fprintln(rw, "OK")
}

func NewHealth(statsd *statsd.Client) *Health {
	return &Health{
		statsd: statsd,
	}
}
