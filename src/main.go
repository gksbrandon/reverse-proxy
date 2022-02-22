package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"expvar"
	"io"
	"io/ioutil"

	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
)

const (
	defaultAPIURL = "https://api.coinbase.com/v2/prices/spot"
)

// DataResponse is the response received from the external API
type DataResponse struct {
	Data struct {
		Base     string `json:"base"`
		Currency string `json:"currency"`
		Amount   string `json:"amount"`
	} `json:"data"`
}

// ErrorResponse is the error response returned to user for a bad request
type ErrorResponse struct {
	Error string
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

// JSONError is a helper function to return a JSON formatted error to the user
func JSONError(rw http.ResponseWriter, err interface{}, code int) {
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.Header().Set("X-Content-Type-Options", "nosniff")
	rw.WriteHeader(code)
	e := json.NewEncoder(rw).Encode(err)
	if e != nil {
		log.Fatal(e)
	}
}

func (a *application) spotPriceHandler(rw http.ResponseWriter, req *http.Request) {
	// Parse apiURL
	apiURL, err := url.Parse(a.apiURL)
	if err != nil {
		JSONError(rw, err, http.StatusBadRequest)
		return
	}

	// Validate currency
	vars := mux.Vars(req)
	currency := vars["currency"]
	acceptedCurrencies := []string{"EUR", "GBP", "USD", "JPY"}
	if ok := contains(acceptedCurrencies, currency); !ok {
		JSONError(rw, ErrorResponse{`Currency not currently supported, please choose between ["EUR", "GBP", "USD", "JPY"]`}, http.StatusBadRequest)
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
		JSONError(rw, err, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read body of response into bytes
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		JSONError(rw, err, http.StatusInternalServerError)
		return
	}

	// Check that server actually sent compressed data
	var reader io.ReadCloser
	var response DataResponse
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		buf := bytes.NewBuffer(bodyBytes)
		reader, err = gzip.NewReader(buf)
		if err != nil {
			JSONError(rw, err, http.StatusInternalServerError)
			return
		}
		if err := json.NewDecoder(reader).Decode(&response); err != nil && err != io.EOF {
			JSONError(rw, err, http.StatusInternalServerError)
			return
		}
	default:
		err = json.Unmarshal(bodyBytes, &response)
		if err != nil {
			JSONError(rw, err, http.StatusInternalServerError)
			return
		}
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
	_, err := io.WriteString(rw, `{"alive": true}`)
	if err != nil {
		log.Fatal(err)
	}
}

func (a *application) handler() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/metrics", expvar.Handler().ServeHTTP).Methods("GET")
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
