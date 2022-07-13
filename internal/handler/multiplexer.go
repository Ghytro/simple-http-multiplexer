package handler

import (
	"bytes"
	"context"
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
	errorMessage        string
	returningStatusCode int
	url                 string
}

func newHttpError(errMessage string, statusCode int, url string) *httpError {
	return &httpError{
		errorMessage:        errMessage,
		returningStatusCode: statusCode,
		url:                 url,
	}
}

func (e httpError) Error() string {
	return fmt.Sprintf("an error occured while accessing url %q: %s", e.url, e.errorMessage)
}

func (e httpError) Url() string {
	return e.url
}

func (e httpError) ReturningStatusCode() int {
	return e.returningStatusCode
}

func performRequests(reqCtx context.Context, urls []string, result chan MuxHandleResponse, chanErr chan error) {
	client := newHttpClient()
	responses := make(chan *http.Response)
	// avoid goroutine leaks with buffered channels for error handling
	errs := make(chan error, config.MaxOutcomingRequestsPerRequest)

	performReq := func(url string) {
		req, err := http.NewRequest("POST", url, strings.NewReader(""))
		if err != nil {
			errs <- err
			return
		}
		ctx, cancel := context.WithTimeout(
			reqCtx,
			time.Second*config.UrlRequestTimeout,
		)
		defer cancel()
		req = req.WithContext(ctx)
		resp, err := client.Do(req)
		if err != nil {
			if strings.Contains(err.Error(), "context deadline exceeded") {
				errStatusCode := http.StatusRequestTimeout
				errMessage := "timeout for request to url"
				errs <- newHttpError(errMessage, errStatusCode, url)
			} else {
				errs <- err
			}
			responses <- nil
			return
		}
		errs <- nil
		responses <- resp
	}
	esResponses := make([]ExternalServiceReponse, 0, len(urls))

	// no need in wg to wait for goroutines, channels do the job
	for i := 0; i < len(urls); i += config.MaxOutcomingRequestsPerRequest {
		for j := 0; i+j < len(urls) && j < config.MaxOutcomingRequestsPerRequest; j++ {
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
	go performRequests(r.Context(), req.Urls, chanMuxResponse, errs)
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
