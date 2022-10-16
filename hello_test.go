package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestMain(m *testing.M) {
	setUp()

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
	req.Header.Set(fasthttp.HeaderAcceptEncoding, "gzip, br")
	resp, err := doRequest(req)

	if err != nil {
		t.Fatalf("request failed: %v", err.Error())
	}
	verifyHelloTypeAndCode(resp, t)
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

func verifyUncompressedHelloResponse(got *fasthttp.Response, wantBody string, t *testing.T) {
	verifyHelloTypeAndCode(got, t)
	gotBody := string(got.Body())
	if gotBody != wantBody {
		t.Errorf("invalid body. Want: %v, got: %v", wantBody, gotBody)
	}
}

func verifyHelloTypeAndCode(got *fasthttp.Response, t *testing.T) {
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
