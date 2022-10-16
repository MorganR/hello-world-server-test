package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/valyala/fasthttp"
)

var (
	baseUrl = flag.String("base_url", "", "The base URL (scheme + host + port), to run integration tests against. Example: http://localhost:80")
	baseUri *fasthttp.URI
	client  *fasthttp.Client
)

func TestMain(m *testing.M) {
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

	os.Exit(m.Run())
}

func TestHello(t *testing.T) {
	uri := getBaseUri()
	uri.SetPath("/hello")

	req := fasthttp.AcquireRequest()
	req.SetURI(uri)
	resp, err := doRequest(req)

	if err != nil {
		t.Fatalf("request failed: %v", err.Error())
	}
	verifyUncompressedHelloResponse(resp, "Hello, world!", t)
}

func TestHelloWithName(t *testing.T) {
	uri := getBaseUri()
	uri.SetPath("/hello")
	args := uri.QueryArgs()
	args.Add("name", `some COOL guy`)

	req := fasthttp.AcquireRequest()
	req.SetURI(uri)
	resp, err := doRequest(req)

	if err != nil {
		t.Fatalf("request failed: %v", err.Error())
	}
	verifyUncompressedHelloResponse(resp, "Hello, some COOL guy!", t)
}

func TestHelloWithEmptyName(t *testing.T) {
	uri := getBaseUri()
	uri.SetPath("/hello")
	args := uri.QueryArgs()
	args.Add("name", "")

	req := fasthttp.AcquireRequest()
	req.SetURI(uri)
	resp, err := doRequest(req)

	if err != nil {
		t.Fatalf("request failed: %v", err.Error())
	}
	verifyUncompressedHelloResponse(resp, "Hello, world!", t)
}

func TestHelloNameMaxLength(t *testing.T) {
	// Max length should succeed.
	uri := getBaseUri()
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
	verifyUncompressedHelloResponse(resp, fmt.Sprintf("Hello, %v!", maxLenName), t)

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
	uri := getBaseUri()
	uri.SetPath("/hello")

	req := fasthttp.AcquireRequest()
	req.SetURI(uri)
	req.Header.Set(fasthttp.HeaderAcceptEncoding, "unknown, br")
	resp, err := doRequest(req)

	if err != nil {
		t.Fatalf("request failed: %v", err.Error())
	}
	verifyHelloTypeAndCode(resp, t)
	gotEncoding := string(resp.Header.ContentEncoding())
	wantEncoding := "br"
	if gotEncoding != wantEncoding {
		t.Fatalf("invalid encoding. Want: %v, got: %v", wantEncoding, gotEncoding)
	}
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

var expectedInvalidPaths = []string{
	"/", "/thing", "/static", "/static/", "/static/no-file-here", "/static/../main.go",
}

func TestInvalidPaths(t *testing.T) {
	for _, p := range expectedInvalidPaths {
		uri := getBaseUri()
		uri.SetPath(p)

		req := fasthttp.AcquireRequest()
		req.SetURI(uri)
		resp := fasthttp.AcquireResponse()
		// Allow redirects (seems to happen with quirks of different routing implementations).
		err := client.DoRedirects(req, resp, 2)

		if err != nil {
			t.Fatalf("request failed for url %v: %v", uri.String(), err.Error())
		}
		gotCode := resp.StatusCode()
		if gotCode < 400 || gotCode > 499 {
			t.Errorf("invalid status code for url %v. Want 4xx, got: %v", uri.String(), gotCode)
		}
	}
}

func TestStaticBasic(t *testing.T) {
	uri := getBaseUri()
	uri.SetPath("/static/basic.html")

	req := fasthttp.AcquireRequest()
	req.SetURI(uri)
	resp, err := doRequest(req)

	if err != nil {
		t.Fatalf("request failed: %v", err.Error())
	}
	verifyTypeAndCode(resp, "text/html; charset=utf-8", http.StatusOK, t)
	gotBody := resp.Body()
	expectedBody, err := os.ReadFile("./data/basic.html")
	if err != nil {
		t.Fatalf("failed to load basic html: %v", err.Error())
	}
	if !cmp.Equal(gotBody, expectedBody) {
		diffs := cmp.Diff(expectedBody, gotBody)
		t.Errorf("incorrect basic html response. Diff: %v", diffs)
	}
}

func TestStaticBasicCompressed(t *testing.T) {
	uri := getBaseUri()
	uri.SetPath("/static/basic.html")

	req := fasthttp.AcquireRequest()
	req.SetURI(uri)
	req.Header.Set(fasthttp.HeaderAcceptEncoding, "unknown, br")
	resp, err := doRequest(req)

	if err != nil {
		t.Fatalf("request failed: %v", err.Error())
	}
	verifyTypeAndCode(resp, "text/html; charset=utf-8", http.StatusOK, t)
	gotBody, err := resp.BodyUnbrotli()
	if err != nil {
		t.Fatalf("failed to uncompress: %v", err.Error())
	}
	expectedBody, err := os.ReadFile("./data/basic.html")
	if err != nil {
		t.Fatalf("failed to load basic html: %v", err.Error())
	}
	if !cmp.Equal(gotBody, expectedBody) {
		diffs := cmp.Diff(expectedBody, gotBody)
		t.Errorf("incorrect basic html response. Diff: %v", diffs)
	}
}

func TestStaticImage(t *testing.T) {
	uri := getBaseUri()
	uri.SetPath("/static/scout.webp")

	req := fasthttp.AcquireRequest()
	req.SetURI(uri)
	req.Header.Set(fasthttp.HeaderAcceptEncoding, "unknown, br")
	resp, err := doRequest(req)

	if err != nil {
		t.Fatalf("request failed: %v", err.Error())
	}
	verifyTypeAndCode(resp, "image/webp", http.StatusOK, t)
	gotBody := resp.Body()
	expectedBody, err := os.ReadFile("./data/scout.webp")
	if err != nil {
		t.Fatalf("failed to load image: %v", err.Error())
	}
	if !cmp.Equal(gotBody, expectedBody) {
		t.Errorf("incorrect image data. Want len: %v, got len: %v", len(expectedBody), len(gotBody))
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

func verifyUncompressedHelloResponse(got *fasthttp.Response, wantBody string, t *testing.T) {
	verifyHelloTypeAndCode(got, t)
	gotBody := string(got.Body())
	if gotBody != wantBody {
		t.Errorf("invalid body. Want: %v, got: %v", wantBody, gotBody)
	}
}

func verifyHelloTypeAndCode(got *fasthttp.Response, t *testing.T) {
	verifyTypeAndCode(got, "text/plain; charset=utf-8", http.StatusOK, t)
}

func verifyTypeAndCode(got *fasthttp.Response, wantType string, wantCode int, t *testing.T) {
	gotType := string(got.Header.ContentType())
	if gotType != wantType {
		t.Errorf("invalid content type. Want: %v, got %v", wantType, gotType)
	}
	gotCode := got.StatusCode()
	if gotCode != wantCode {
		t.Errorf("invalid status code. Want: %v, got: %v", wantCode, gotCode)
	}
}
