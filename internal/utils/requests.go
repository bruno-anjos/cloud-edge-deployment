package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	ReqIDHeaderField = "REQ_ID"
)

func BuildRequest(method, host, path string, body interface{}) *http.Request {
	hostURL := url.URL{
		Scheme:      "http",
		Opaque:      "",
		User:        nil,
		Host:        host,
		Path:        path,
		RawPath:     "",
		ForceQuery:  false,
		RawQuery:    "",
		Fragment:    "",
		RawFragment: "",
	}

	var (
		err        error
		request    *http.Request
		bodyBuffer *bytes.Buffer
	)

	if body != nil {
		var jsonStr []byte

		jsonStr, err = json.Marshal(body)
		if err != nil {
			panic(err)
		}

		bodyBuffer = bytes.NewBuffer(jsonStr)
	} else {
		bodyBuffer = new(bytes.Buffer)
	}

	request, err = http.NewRequestWithContext(context.Background(), method, hostURL.String(), bodyBuffer)
	if err != nil {
		panic(err)
	}

	request.Header.Set("Content-Type", "application/json")

	return request
}

func DoRequest(httpClient *http.Client, request *http.Request, responseBody interface{}) (status int, timedOut bool) {
	log.Debugf("Doing request: %s %s", request.Method, request.URL.String())

	if httpClient == nil {
		panic(errorHTTPClietNilFormat)
	}

	reqID, err := uuid.NewUUID()
	if err != nil {
		panic(err)
	}

	request.Header.Add(ReqIDHeaderField, reqID.String())

	resp, err := httpClient.Do(request)
	if err != nil {
		status = -1

		timedOut = os.IsTimeout(err)

		log.Warn(err)

		return
	}

	if responseBody != nil {
		wantsResponseBody(resp, responseBody)
	} else {
		ignoreResponse(resp)
	}

	log.Debugf("Done: %s %s", request.Method, request.URL.String())

	status = resp.StatusCode

	return status, timedOut
}

func wantsResponseBody(resp *http.Response, responseBody interface{}) {
	err := json.NewDecoder(resp.Body).Decode(responseBody)
	if err != nil {
		panic(err)
	}

	err = resp.Body.Close()
	if err != nil {
		panic(err)
	}
}

func ignoreResponse(resp *http.Response) {
	_, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	err = resp.Body.Close()
	if err != nil {
		panic(err)
	}
}

func ExtractPathVar(r *http.Request, varName string) (varValue string) {
	vars := mux.Vars(r)

	var ok bool
	varValue, ok = vars[varName]

	if !ok {
		err := errors.Errorf("var %s was not in request path", varName)
		panic(err)
	}

	return
}
