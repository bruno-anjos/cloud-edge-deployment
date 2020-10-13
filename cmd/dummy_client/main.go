package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	defaultHttp "net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	log "github.com/sirupsen/logrus"

	"github.com/bruno-anjos/archimedesHTTPClient"
)

const (
	invalid = "__invalid"
)

type config struct {
	Service         string               `json:"service"`
	RequestTimeout  int                  `json:"request_timeout"`
	MaxRequests     int                  `json:"max_requests"`
	NumberOfClients int                  `json:"number_of_clients"`
	Fallback        string               `json:"fallback"`
	Location        publicUtils.Location `json:"location"`
	Port            int                  `json:"port"`
}

func main() {
	debug := flag.Bool("d", false, "enable debug logs")
	configFilename := flag.String("config", invalid, "configuration file name")
	flag.Parse()

	if *configFilename == invalid {
		log.Fatalf("config file name missing")
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
		log.Fatalf("port is zero")
	}

	serviceUrl := url.URL{
		Scheme: "http",
		Host:   conf.Service + ":" + strconv.Itoa(conf.Port),
	}

	wg := &sync.WaitGroup{}
	log.Debugf("Launching %d clients...", conf.NumberOfClients)
	for i := 1; i <= conf.NumberOfClients; i++ {
		wg.Add(1)
		go runClient(wg, i, serviceUrl, &conf)
	}

	wg.Wait()
}

func runClient(wg *sync.WaitGroup, clientNum int, serviceUrl url.URL, config *config) {
	defer wg.Done()

	log.Debugf("[%d] Starting client", clientNum)

	client := &http.Client{}
	client.InitArchimedesClient(config.Fallback, archimedes.Port, &config.Location)
	r, err := http.NewRequest(defaultHttp.MethodGet, serviceUrl.String(), nil)
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
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Errorf("[%d] got status %d", clientNum, resp.StatusCode)
	}
}
