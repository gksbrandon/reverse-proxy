package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

func main() {
	demoURL, err := url.Parse("https://api.coinbase.com/v2/prices/spot?currency=USD")
	fmt.Println(demoURL.Host, demoURL.Scheme, demoURL.Path, demoURL.RawQuery)
	if err != nil {
		log.Fatal(err)
	}

	proxy := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		req.Host = demoURL.Host
		req.URL.Host = demoURL.Host
		req.URL.Scheme = demoURL.Scheme
		req.URL.Path = demoURL.Path
		req.URL.RawQuery = demoURL.RawQuery
		req.RequestURI = ""
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, err)
			return
		}
		rw.WriteHeader(resp.StatusCode)
		io.Copy(rw, resp.Body)
	})

	http.ListenAndServe(":9000", proxy)
}
