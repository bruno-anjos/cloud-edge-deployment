package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	defaultHttp "net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	http "github.com/bruno-anjos/archimedesHTTPClient"

	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"
)

const (
	invalid               = "__invalid"
	maxTimeBetweenClients = 120
)

type config struct {
	Deployment      string      `json:"service"`
	RequestTimeout  int         `json:"request_timeout"`
	MaxRequests     int         `json:"max_requests"`
	NumberOfClients int         `json:"number_of_clients"`
	Fallback        *utils.Node `json:"fallback"`
	Location        struct {
		Lat float64
		Lng float64
	} `json:"location"`
	Port int `json:"port"`
}

func main() {
	debug := flag.Bool("d", false, "enable debug logs")
	configFilename := flag.String("config", invalid, "configuration file name")
	flag.Parse()

	if *configFilename == invalid {
		log.Panic("config file name missing")
	}

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	configBytes, err := ioutil.ReadFile(*configFilename)
	if err != nil {
		panic(err)
	}

	var conf config

	err = json.Unmarshal(configBytes, &conf)
	if err != nil {
		panic(err)
	}

	if conf.Port == 0 {
		log.Panic("port is zero")
	}

	location := s2.LatLngFromDegrees(conf.Location.Lat, conf.Location.Lng)

	deploymentURL := url.URL{
		Scheme:      "http",
		Opaque:      "",
		User:        nil,
		Host:        conf.Deployment + ":" + strconv.Itoa(conf.Port),
		Path:        "",
		RawPath:     "",
		ForceQuery:  false,
		RawQuery:    "",
		Fragment:    "",
		RawFragment: "",
	}
	wg := &sync.WaitGroup{}

	log.Debugf("Launching %d clients with config %+v", conf.NumberOfClients, conf)

	for i := 1; i <= conf.NumberOfClients; i++ {
		wg.Add(1)

		go runClient(wg, i, deploymentURL, &conf, location)
	}

	wg.Wait()
}

func runClient(wg *sync.WaitGroup, clientNum int, deploymentURL url.URL, config *config, location s2.LatLng) {
	defer wg.Done()

	waitTime := time.Duration(internalUtils.GetRandInt(maxTimeBetweenClients)) * time.Second
	time.Sleep(waitTime)

	log.Debugf("[%d] Starting client", clientNum)

	client := &http.Client{}
	client.InitArchimedesClient(config.Fallback.Addr, archimedes.Port, location)

	r, err := http.NewRequestWithContext(context.Background(), defaultHttp.MethodGet, deploymentURL.String(), nil)
	if err != nil {
		panic(err)
	}

	ticker := time.NewTicker(time.Duration(config.RequestTimeout) * time.Second)

	for i := 0; i < config.MaxRequests; i++ {
		doRequest(client, r, clientNum)
		<-ticker.C
	}
}

func doRequest(client *http.Client, r *http.Request, clientNum int) {
	resp, err := client.Do(r)
	if err != nil {
		log.Panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Errorf("[%d] got status %d", clientNum, resp.StatusCode)
	}

	err = resp.Body.Close()
	if err != nil {
		log.Panic(err)
	}
}
