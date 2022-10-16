package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

var baseUrl = flag.String("base_url", "", "The base URL (scheme + host + port), to run integration tests against. Example: http://localhost:80")
var baseUri *fasthttp.URI

var client = &fasthttp.Client{
	Name: "integration-tester",
}

func TestMain(m *testing.M) {
	flag.Parse()

	var err error
	baseUri = fasthttp.AcquireURI()
	err = baseUri.Parse(nil, []byte(*baseUrl))
	if err != nil {
		log.Fatalf("Could not parse base URL (%v): %v", *baseUrl, err.Error())
	}

	os.Exit(m.Run())
}

func TestHello(t *testing.T) {
	uri := fasthttp.AcquireURI()
	baseUri.CopyTo(uri)
	uri.SetPath("/hello")

	req := fasthttp.AcquireRequest()
	req.SetURI(uri)
	resp, err := doRequest(req)

	if err != nil {
		t.Fatalf("request failed: %v", err.Error())
	}
	verifyUncompressedResponse(resp, "Hello, world!", t)
}

func TestHelloWithName(t *testing.T) {
	uri := fasthttp.AcquireURI()
	baseUri.CopyTo(uri)
	uri.SetPath("/hello")
	args := uri.QueryArgs()
	args.Add("name", `some COOL guy`)

	req := fasthttp.AcquireRequest()
	req.SetURI(uri)
	resp, err := doRequest(req)

	if err != nil {
		t.Fatalf("request failed: %v", err.Error())
	}
	verifyUncompressedResponse(resp, "Hello, some COOL guy!", t)
}

func TestHelloWithEmptyName(t *testing.T) {
	uri := fasthttp.AcquireURI()
	baseUri.CopyTo(uri)
	uri.SetPath("/hello")
	args := uri.QueryArgs()
	args.Add("name", "")

	req := fasthttp.AcquireRequest()
	req.SetURI(uri)
	resp, err := doRequest(req)

	if err != nil {
		t.Fatalf("request failed: %v", err.Error())
	}
	verifyUncompressedResponse(resp, "Hello, world!", t)
}

func TestHelloNameMaxLength(t *testing.T) {
	// Max length should succeed.
	uri := fasthttp.AcquireURI()
	baseUri.CopyTo(uri)
	uri.SetPath("/hello")
	args := uri.QueryArgs()
	maxLenName := strings.Repeat("a", 100)
	args.Set("name", maxLenName)

	req := fasthttp.AcquireRequest()
	req.SetURI(uri)
	resp, err := doRequest(req)

	if err != nil {
		t.Fatalf("max len name request failed: %v", err.Error())
	}
	verifyUncompressedResponse(resp, fmt.Sprintf("Hello, %v!", maxLenName), t)

	// Too long should fail.
	args.Set("name", maxLenName+"a")

	req.Reset()
	req.SetURI(uri)
	resp, err = doRequest(req)

	if err != nil {
		t.Fatalf("name too long request failed: %v", err.Error())
	}
	gotCode := resp.StatusCode()
	wantCode := http.StatusBadRequest
	if gotCode != wantCode {
		t.Errorf("invalid status code for name too long. Want: %v, got: %v", wantCode, gotCode)
	}
}

func TestHelloCompression(t *testing.T) {
	uri := fasthttp.AcquireURI()
	baseUri.CopyTo(uri)
	uri.SetPath("/hello")

	req := fasthttp.AcquireRequest()
	req.SetURI(uri)
	req.Header.Set(fasthttp.HeaderAcceptEncoding, "gzip, br")
	resp, err := doRequest(req)

	if err != nil {
		t.Fatalf("request failed: %v", err.Error())
	}
	verifyTypeAndCode(resp, t)
	uncompressed, err := resp.BodyUnbrotli()
	if err != nil {
		t.Fatalf("failed to uncompress: %v", err.Error())
	}
	gotBody := string(uncompressed)
	wantBody := "Hello, world!"
	if gotBody != wantBody {
		t.Errorf("invalid body. Want: %v, got: %v", wantBody, gotBody)
	}
}

func doRequest(r *fasthttp.Request) (*fasthttp.Response, error) {
	resp := fasthttp.AcquireResponse()
	err := client.Do(r, resp)
	return resp, err
}

func verifyUncompressedResponse(got *fasthttp.Response, wantBody string, t *testing.T) {
	verifyTypeAndCode(got, t)
	gotBody := string(got.Body())
	if gotBody != wantBody {
		t.Errorf("invalid body. Want: %v, got: %v", wantBody, gotBody)
	}
}

func verifyTypeAndCode(got *fasthttp.Response, t *testing.T) {
	gotType := string(got.Header.ContentType())
	wantType := "text/plain"
	if gotType != wantType {
		t.Errorf("invalid content type. Want: %v, got %v", wantType, gotType)
	}
	gotCode := got.StatusCode()
	wantCode := http.StatusOK
	if gotCode != wantCode {
		t.Errorf("invalid status code. Want: %v, got: %v", wantCode, gotCode)
	}
}
