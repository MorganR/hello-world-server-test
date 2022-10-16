package main

import (
	"flag"
	"log"

	"github.com/valyala/fasthttp"
)

var (
	baseUrl = flag.String("base_url", "", "The base URL (scheme + host + port), to run integration tests against. Example: http://localhost:80")
	baseUri *fasthttp.URI
	client  *fasthttp.Client
)

func setUp() {
	flag.Parse()

	if *baseUrl == "" {
		log.Fatal("Must provide a valid base_url")
	}
	baseUri = fasthttp.AcquireURI()
	err := baseUri.Parse(nil, []byte(*baseUrl))
	if err != nil {
		log.Fatalf("Could not parse base URL (%v): %v", *baseUrl, err.Error())
	}

	client = &fasthttp.Client{
		Name: "integration-tester",
	}
}

func getBaseUri() *fasthttp.URI {
	uri := fasthttp.AcquireURI()
	baseUri.CopyTo(uri)
	return uri
}

func doRequest(r *fasthttp.Request) (*fasthttp.Response, error) {
	resp := fasthttp.AcquireResponse()
	err := client.Do(r, resp)
	return resp, err
}
