package test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Ghytro/simple-http-multiplexer/internal/handler"
)

// not really a test, just for logging
func TestMuxHandler(t *testing.T) {
	reqJson := `
	{
		"urls": [
			"https://example.com",
			"https://example.org",
			"https://example.net",
			"https://example.edu/"
		]
	}
	`
	req, err := http.NewRequest(
		"POST",
		"http://localhost:8080/api/mux",
		strings.NewReader(reqJson),
	)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	http.HandlerFunc(handler.MuxHandler).ServeHTTP(rr, req)
	if rr.Result().StatusCode != http.StatusOK {
		t.Fatalf(
			"an error occured while running handler, return code: %d, body: %s",
			rr.Result().StatusCode,
			rr.Body.String(),
		)
	}

	var request handler.MuxHandleRequest
	if err := json.Unmarshal([]byte(reqJson), &request); err != nil {
		t.Fatal(err)
	}
	responses := make([]*http.Response, len(request.Urls))
	errs := make(chan error, len(request.Urls))
	var wg sync.WaitGroup
	wg.Add(len(request.Urls))
	for i, u := range request.Urls {
		go func(idx int, url string) {
			resp, err := http.Post(url, "", strings.NewReader(""))
			if err != nil {
				errs <- err
				return
			}
			responses[idx] = resp
			wg.Done()
		}(i, u)
	}
	wg.Wait()
	if len(errs) != 0 {
		t.Fatal(<-errs)
	}

	var handlerResponse handler.MuxHandleResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &handlerResponse); err != nil {
		t.Fatal(err)
	}
	for _, r1 := range responses {
		got := handler.ExternalServiceReponse{}
		for _, r2 := range handlerResponse.Responses {
			if r1.Request.URL.String() == r2.ServiceUrl {
				got = r2
				break
			}
		}
		if got.ServiceUrl == "" {
			t.Fatalf("not found response for service %s", r1.Request.URL.String())
		}
		buf := bytes.Buffer{}
		base64Encoder := base64.NewEncoder(base64.StdEncoding, &buf)
		if _, err := io.Copy(base64Encoder, r1.Body); err != nil {
			t.Fatal(err)
		}
		base64Encoder.Close()
		if buf.String() != got.Base64Payload {
			t.Fatalf("payloads for request to service %s don't match", got.ServiceUrl)
		}
	}
}
