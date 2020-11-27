package utils

import (
	"flag"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// LocalhostAddr contains the default interface address
	LocalhostAddr = "0.0.0.0"
)

// StartServer seeds the random generator and starts a server on the
// specified host and port serving the routes passed with a specified prefix.
func StartServer(deploymentName string, port int, prefixPath string, routes []Route) {
	rand.Seed(time.Now().UnixNano())

	debug := flag.Bool("d", false, "add debug logs")
	listenAddr := flag.String("l", LocalhostAddr, "address to listen on")
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Debug("starting log in debug mode")
	r := NewRouter(prefixPath, routes)

	var listenAddrPort string
	if *listenAddr != "" {
		listenAddrPort = *listenAddr + ":" + strconv.Itoa(port)
	} else {
		listenAddrPort = LocalhostAddr
	}

	log.Infof("%s server listening at %s...\n", deploymentName, listenAddrPort)
	log.Panic(http.ListenAndServe(listenAddrPort, r))
}

func StartServerWithoutDefaultFlags(deploymentName string, port int, prefixPath string, routes []Route,
	debug *bool, listenAddr *string) {
	rand.Seed(time.Now().UnixNano())

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Debug("starting log in debug mode")
	r := NewRouter(prefixPath, routes)

	var listenAddrPort string
	if *listenAddr != "" {
		listenAddrPort = *listenAddr + ":" + strconv.Itoa(port)
	} else {
		listenAddrPort = LocalhostAddr
	}

	log.Infof("%s server listening at %s...\n", deploymentName, listenAddrPort)
	log.Panic(http.ListenAndServe(listenAddrPort, r))
}
