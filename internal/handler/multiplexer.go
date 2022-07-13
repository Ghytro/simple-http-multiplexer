package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Ghytro/simple-http-multiplexer/internal/config"
	"github.com/Ghytro/simple-http-multiplexer/pkg/algorithm"
)

type MuxHandleRequest struct {
	Urls []string `json:"urls"`
}

type MuxHandleResponse struct {
	Responses []ExternalServiceReponse `json:"responses"`
}

type ExternalServiceReponse struct {
	ServiceUrl     string `json:"service_url"`
	HttpStatusCode int    `json:"http_status_code"`
	Base64Payload  string `json:"base64_payload"`
	ContentType    string `json:"content_type"`
}

func isIncorrectUrl(u string) bool {
	_, err := url.ParseRequestURI(u)
	return err != nil
}

func newHttpClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * config.UrlRequestTimeout,
		Transport: &http.Transport{
			MaxIdleConns: config.MaxOutcomingRequestsPerRequest,
		},
	}
}

func makeExternalServiceResponse(resp *http.Response) (ExternalServiceReponse, error) {
	esResponse := ExternalServiceReponse{
		ServiceUrl:     resp.Request.URL.String(),
		HttpStatusCode: resp.StatusCode,
		ContentType:    resp.Header.Get("Content-Type"),
	}
	base64ResponseWriter := bytes.Buffer{}
	base64Encoder := base64.NewEncoder(base64.StdEncoding, &base64ResponseWriter)
	if _, err := io.Copy(base64Encoder, resp.Body); err != nil {
		return ExternalServiceReponse{}, err
	}
	base64Encoder.Close()
	esResponse.Base64Payload = base64ResponseWriter.String()
	return esResponse, nil
}

type httpError struct {
	err                 error
	returningStatusCode int
	url                 string
}

func newHttpError(err error, statusCode int, url string) *httpError {
	return &httpError{
		err:                 err,
		returningStatusCode: statusCode,
		url:                 url,
	}
}

func (e httpError) Error() string {
	return fmt.Sprintf("an error occured while accessing url %s: %s", e.url, e.err)
}

func (e httpError) Err() error {
	return e.err
}

func (e httpError) Url() string {
	return e.url
}

func (e httpError) ReturningStatusCode() int {
	return e.returningStatusCode
}

func performRequests(urls []string, result chan MuxHandleResponse, chanErr chan error) {
	client := newHttpClient()
	responses := make(chan *http.Response)
	// avoid goroutine leaks with buffered channels for error handling
	errs := make(chan error, config.MaxOutcomingRequestsPerRequest)

	performReq := func(url string) {
		resp, err := client.Post(url, "", strings.NewReader(""))
		if err != nil {
			errStatusCode := http.StatusInternalServerError
			errMessage := err.Error()
			if strings.Contains(err.Error(), "context deadline exceeded") {
				errStatusCode = http.StatusRequestTimeout
				errMessage = "timeout for request to url"
			}
			errs <- newHttpError(err, errStatusCode, errMessage)
			responses <- nil
			return
		}
		errs <- nil
		responses <- resp
	}
	esResponses := make([]ExternalServiceReponse, 0, len(urls))

	// no need in wg to wait for goroutines, channels do the job
	for i := 0; i < len(urls); i += config.MaxOutcomingRequestsPerRequest {
		for j := 0; i+j < len(urls); j++ {
			go performReq(urls[i+j])
		}
		// handle all the errors first
		for j := 0; j < config.MaxOutcomingRequestsPerRequest; j++ {
			err := <-errs
			if err != nil {
				chanErr <- err
				return
			}
		}
		for j := 0; j < config.MaxOutcomingRequestsPerRequest; j++ {
			resp := <-responses
			esResponse, err := makeExternalServiceResponse(resp)
			if err != nil {
				chanErr <- err
				return
			}
			esResponses = append(esResponses, esResponse)
		}
	}
	result <- MuxHandleResponse{esResponses}
}

func MuxHandler(w http.ResponseWriter, r *http.Request) {
	req := &MuxHandleRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		http.Error(
			w,
			"Expected json encoded data",
			http.StatusBadRequest,
		)
		return
	}
	if len(req.Urls) > config.MaxUrlPerRequest {
		http.Error(
			w,
			fmt.Sprintf(
				"Too many urls requested, got %d, but max is: %d",
				len(req.Urls),
				config.MaxUrlPerRequest,
			),
			http.StatusBadRequest,
		)
		return
	}
	if incUrlIdx := algorithm.FindIf(req.Urls, isIncorrectUrl); incUrlIdx != -1 {
		http.Error(
			w,
			"Incorrect format of incoming url: "+req.Urls[incUrlIdx],
			http.StatusBadRequest,
		)
		return
	}
	chanMuxResponse := make(chan MuxHandleResponse, 1)
	errs := make(chan error, 1)
	go performRequests(req.Urls, chanMuxResponse, errs)
	select {
	case muxResponse := <-chanMuxResponse:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(muxResponse)
	case err := <-errs:
		if httpErr, ok := err.(httpError); ok {
			http.Error(
				w,
				err.Error(),
				httpErr.returningStatusCode,
			)
			break
		}
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
	case <-time.After(time.Second * config.RequestHandleTimeout):
		http.Error(
			w,
			"Request timeout",
			http.StatusRequestTimeout,
		)
	case <-r.Context().Done():
		break
	}
}
