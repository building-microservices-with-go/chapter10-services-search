package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/building-microservices-with-go/chapter10-services-search/data"
	"github.com/stretchr/testify/assert"
)

var mockStore *data.MockStore

func TestSearchHandlerReturnsBadRequestWhenNoSearchCriteriaIsSent(t *testing.T) {
	r, rw, handler := setupTest(nil)

	handler.Handle(rw, r)

	if rw.Code != http.StatusBadRequest {
		t.Errorf("Expected BadRequest got %v", rw.Code)
	}
}

func TestSearchHandlerReturnsBadRequestWhenBlankSearchCriteriaIsSent(t *testing.T) {
	r, rw, handler := setupTest(&searchRequest{})

	handler.Handle(rw, r)

	if rw.Code != http.StatusBadRequest {
		t.Errorf("Expected BadRequest got %v", rw.Code)
	}
}

func TestSearchHandlerCallsDataStoreWithValidQuery(t *testing.T) {
	r, rw, handler := setupTest(&searchRequest{Query: "Fat Freddy's Cat"})
	mockStore.On("Search", "Fat Freddy's Cat").Return(make([]data.Kitten, 0))

	handler.Handle(rw, r)

	mockStore.AssertExpectations(t)
}

func TestSearchHandlerReturnsKittensWithValidQuery(t *testing.T) {
	r, rw, handler := setupTest(&searchRequest{Query: "Fat Freddy's Cat"})
	mockStore.On("Search", "Fat Freddy's Cat").Return(make([]data.Kitten, 1))

	handler.Handle(rw, r)

	response := searchResponse{}
	json.Unmarshal(rw.Body.Bytes(), &response)

	assert.Equal(t, 1, len(response.Kittens))
	assert.Equal(t, http.StatusOK, rw.Code)
}

func setupTest(d interface{}) (*http.Request, *httptest.ResponseRecorder, *Search) {
	mockStore = &data.MockStore{}

	statsdClient, _ := statsd.New("127.0.0.1:8125")

	h := NewSearch(mockStore, statsdClient)

	rw := httptest.NewRecorder()

	if d == nil {
		return httptest.NewRequest("POST", "/search", nil), rw, h
	}

	body, _ := json.Marshal(d)
	return httptest.NewRequest("POST", "/search", bytes.NewReader(body)), rw, h
}
