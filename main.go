package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"

	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
)

const (
	defaultAPIURL = "https://api.coinbase.com/v2/prices/spot"
)

type DataResponse struct {
	Data struct {
		Base     string `json:"base"`
		Currency string `json:"currency"`
		Amount   string `json:"amount"`
	} `json:"data"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type application struct {
	apiURL string
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func (a *application) spotPriceHandler(rw http.ResponseWriter, req *http.Request) {
	// Parse apiURL
	apiURL, err := url.Parse(a.apiURL)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(rw).Encode(ErrorResponse{fmt.Sprintf("Invalid URL: %v, Error: %v", a.apiURL, err)})
		return
	}

	// Validate currency
	vars := mux.Vars(req)
	currency := vars["currency"]
	acceptedCurrencies := []string{"EUR", "GBP", "USD", "JPY"}
	if ok := contains(acceptedCurrencies, currency); !ok {
		rw.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(rw).Encode(ErrorResponse{`Currency not accepted, please choose between "EUR, GBP, USD, JPY"`})
		return
	}

	// Copy request
	rawQuery := "currency=" + currency
	req.Host = apiURL.Host
	req.URL.Host = apiURL.Host
	req.URL.Scheme = apiURL.Scheme
	req.URL.Path = apiURL.Path
	req.URL.RawQuery = rawQuery
	req.RequestURI = ""
	req.Header.Add("Accept-Encoding", "gzip")

	// Make request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(rw, err)
		return
	}
	defer resp.Body.Close()

	// Read body of response into bytes
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(rw, err)
		return
	}

	// Check that server actually sent compressed data
	buf := bytes.NewBuffer(bodyBytes)
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(buf)
		if err != nil {
			fmt.Println("test")
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, err)
			return
		}
	default:
		reader = resp.Body
	}

	// Decode json from the io.Reader
	var response DataResponse
	if err := json.NewDecoder(reader).Decode(&response); err != nil && err != io.EOF {
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(rw, err)
		return
	}

	// Write response
	rw.WriteHeader(resp.StatusCode)
	err = json.NewEncoder(rw).Encode(response)
	if err != nil {
		log.Fatal(err)
	}

}

func (*application) healthCheckHandler(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
	rw.Header().Set("Content-Type", "application/json")
	io.WriteString(rw, `{"alive": true}`)
}

func (a *application) handler() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/health", http.HandlerFunc(a.healthCheckHandler)).Methods("GET")
	r.HandleFunc("/{currency}", http.HandlerFunc(a.spotPriceHandler)).Methods("GET")
	return r
}

func main() {
	a := application{
		apiURL: defaultAPIURL,
	}
	r := a.handler()

	log.Println("Server starting on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
