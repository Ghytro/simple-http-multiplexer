package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

func handleUnknownError(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(
		w,
		"Unknown error occured, try again later",
		http.StatusInternalServerError,
	)
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
	esResponse.Base64Payload = base64ResponseWriter.String()
	return esResponse, nil
}

func MuxHandler(w http.ResponseWriter, r *http.Request) {
	req := &MuxHandleRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		handleUnknownError(w, err)
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

	client := newHttpClient()
	errs := make(chan error)
	responses := make(chan *http.Response)

	performReq := func(url string) {
		resp, err := client.Post(url, "", strings.NewReader(""))
		if err != nil {
			errs <- fmt.Errorf("error accessing url %s: %s", url, err)
			responses <- nil
			return
		}
		responses <- resp
		errs <- nil
	}
	esResponses := make([]ExternalServiceReponse, 0, len(req.Urls))

	// no need in wg to wait for goroutines, channels do the job
	for i := 0; i < len(req.Urls); i += config.MaxOutcomingRequestsPerRequest {
		for j := 0; i+j < len(req.Urls); j++ {
			go performReq(req.Urls[i+j])
		}
		for j := 0; j < config.MaxOutcomingRequestsPerRequest; j++ {
			resp := <-responses
			// if response was nil, the error occured
			// so the channel is read until getting non nil error
			if resp == nil {
				err := <-errs
				for err == nil {
					err = <-errs
				}
				http.Error(
					w,
					err.Error(),
					http.StatusInternalServerError,
				)
				return
			}
			esResponse, err := makeExternalServiceResponse(resp)
			if err != nil {
				handleUnknownError(w, err)
				return
			}
			esResponses = append(esResponses, esResponse)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MuxHandleResponse{esResponses})
}
