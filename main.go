// Service registry keeps track of all services in the network.
//
// It makes mDNS queries to get information about how many origins
// and stream publishers are there in the network. And then it constructs
// urls for the clients to access in the home page.
//
// The no. of urls are a n*m product
// where n = no. of origins and,
// m = no. of cache servers
//
// The list is re-queried on a specified interval.
//
// The client can choose to access any of these urls depending on where
// they are located. The origin info is sent to the cache server in a
// query string from which it knows which origin to point to.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	host          string
	port          string
	dir           string
	iface         string
	queryInterval string
)

func main() {
	flag.StringVar(&host, "host", "", "host address to bind to")
	flag.StringVar(&port, "port", "8080", "listening port")
	flag.StringVar(&dir, "dir", "", "directory path to serve")
	flag.StringVar(&iface, "iface", "wlp4s0", "interface to publish service info")
	flag.StringVar(&queryInterval, "queryInterval", "5m", `interval after which to re-query the list of services. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".`)
	flag.Parse()

	logger := log.New(os.Stdout, "[serve] ", log.LstdFlags|log.Lshortfile)

	if p, ok := os.LookupEnv("PORT"); ok {
		port = p
	}

	duration, err := time.ParseDuration(queryInterval)
	if err != nil {
		logger.Fatal(err)
	}

	registry := NewRegistry(dir, host, port, duration, logger)
	registry.Start()

	// listen for signals
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	logger.Printf("Started server at %s\n", port)

	// Block until one of the signals above is received
	<-signalCh
	logger.Println("Quit signal received, initializing shutdown...")
	logger.Println("Stopping registry")
	registry.Stop()
}
