package utils

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/url"

	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func resolve(toResolve string) (resolved string) {
	host, port, err := net.SplitHostPort(toResolve)
	if err != nil {
		panic(err)
	}

	resolved = toResolve

	switch host {
	case publicUtils.ArchimedesServiceName, publicUtils.DeployerServiceName, publicUtils.SchedulerServiceName,
		publicUtils.AutonomicServiceName:
		resolved = "localhost" + ":" + port
	}

	return
}

func BuildRequest(method, host, path string, body interface{}) *http.Request {
	hostUrl := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   path,
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

	request, err = http.NewRequest(method, hostUrl.String(), bodyBuffer)
	if err != nil {
		panic(err)
	}

	request.Header.Set("Content-Type", "application/json")

	return request
}

func DoRequest(httpClient *http.Client, request *http.Request, responseBody interface{}) (int, *http.Response) {
	request.URL.Host = resolve(request.URL.Host)

	log.Debugf("Doing request: %s %s", request.Method, request.URL.String())

	if httpClient == nil {
		panic(errorHttpClietNilFormat)
	}

	resp, err := httpClient.Do(request)
	if err != nil {
		return -1, nil
	}

	if responseBody != nil {
		err = json.NewDecoder(resp.Body).Decode(responseBody)
		if err != nil {
			panic(err)
		}
	}

	return resp.StatusCode, resp
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
