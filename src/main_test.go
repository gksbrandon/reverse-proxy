package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestHealthCheckHandler(t *testing.T) {
	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/health-check", nil)
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rec := httptest.NewRecorder()

	a := application{
		apiURL: defaultAPIURL,
	}

	// Call ServeHTTP method to directly and pass in our Request and ResponseRecorder.
	handler := http.HandlerFunc(a.healthCheckHandler)
	handler.ServeHTTP(rec, req)

	// Check the status code is what we expect.
	if status := rec.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `{"alive": true}`
	if rec.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rec.Body.String(), expected)
	}
}

func TestSpotPriceHandler(t *testing.T) {

	t.Run("Happy path", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			_, err := rw.Write([]byte(`{"data":{"base":"BTC","currency":"USD","amount":"44259.64"}}`))
			require.NoError(t, err)
		}))
		defer testServer.Close()

		req, err := http.NewRequest("GET", "/:currency", nil)
		if err != nil {
			t.Fatalf("Could not create request: %v", err)
		}

		vars := map[string]string{
			"currency": "USD",
		}

		req = mux.SetURLVars(req, vars)
		rec := httptest.NewRecorder()

		a := application{
			apiURL: testServer.URL,
		}

		handler := http.HandlerFunc(a.spotPriceHandler)
		handler.ServeHTTP(rec, req)
		// Check the status code is what we expect.
		if status := rec.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response DataResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil && err != io.EOF {
			t.Errorf("Expected json decoding success; got %v", err)
		}
		if response.Data.Currency != "USD" {
			t.Errorf("Expected USD; got %v", response.Data.Currency)
		}
	})

	t.Run("Currency not implemented", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			_, err := rw.Write([]byte(`{"data":{"base":"BTC","currency":"USD","amount":"44259.64"}}`))
			require.NoError(t, err)
		}))
		defer testServer.Close()

		req, err := http.NewRequest("GET", "/:currency", nil)
		if err != nil {
			t.Fatalf("Could not create request: %v", err)
		}

		vars := map[string]string{
			"currency": "MYR",
		}

		req = mux.SetURLVars(req, vars)
		rec := httptest.NewRecorder()

		a := application{
			apiURL: testServer.URL,
		}

		handler := http.HandlerFunc(a.spotPriceHandler)
		handler.ServeHTTP(rec, req)

		// Check the status code is what we expect.
		if status := rec.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}

		var response ErrorResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil && err != io.EOF {
			t.Errorf("Expected json decoding success; got %v", err)
		}
		if response.Error != `Currency not currently supported, please choose between ["EUR", "GBP", "USD", "JPY"]` {
			t.Errorf("Expected USD; got %v", response.Error)
		}
	})

	t.Run("Invalid URL", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			_, err := rw.Write([]byte(`{"data":{"base":"BTC","currency":"USD","amount":"44259.64"}}`))
			require.NoError(t, err)
		}))
		defer testServer.Close()

		req, err := http.NewRequest("GET", "/:currency", nil)
		if err != nil {
			t.Fatalf("Could not create request: %v", err)
		}

		vars := map[string]string{
			"currency": "MYR",
		}

		req = mux.SetURLVars(req, vars)
		rec := httptest.NewRecorder()

		a := application{
			apiURL: "%_+",
		}

		handler := http.HandlerFunc(a.spotPriceHandler)
		handler.ServeHTTP(rec, req)

		// Check the status code is what we expect.
		if status := rec.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})
}

func TestRouting(t *testing.T) {
	a := application{
		apiURL: defaultAPIURL,
	}
	srv := httptest.NewServer(a.handler())
	defer srv.Close()

	res, err := http.Get(fmt.Sprintf("%s/USD", srv.URL))
	if err != nil {
		t.Fatalf("Could not send GET request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", res.Status)
	}
}
