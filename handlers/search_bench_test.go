package handlers

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/building-microservices-with-go/chapter10-services-search/data"
)

func BenchmarkSearchHandler(b *testing.B) {
	mockStore = &data.MockStore{}
	mockStore.On("Search", "Fat Freddy's Cat").Return([]data.Kitten{
		data.Kitten{
			Name: "Fat Freddy's Cat",
		},
	})

	statsdClient, _ := statsd.New("127.0.0.1:8125")
	search := NewSearch(mockStore, statsdClient)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r := httptest.NewRequest("POST", "/search", bytes.NewReader([]byte(`{"query":"Fat Freddy's Cat"}`)))
		rr := httptest.NewRecorder()
		search.Handle(rr, r)
	}

}
