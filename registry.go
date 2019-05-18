package main

import (
	"context"
	"html/template"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/micro/mdns"
	"github.com/syntaqx/serve"
)

type registry struct {
	servicesMu          sync.RWMutex
	serviceMap          map[string][]*mdns.ServiceEntry
	resetServiceMapChan chan struct{} // the serviceMap will be emptied every time
	// this signal is fired. Essentially this is done for every new query.
	srv                     *http.Server
	streamTmpl, homeTmpl    *template.Template
	logger                  *log.Logger
	queryInterval           time.Duration
	entriesChan             chan *mdns.ServiceEntry
	quitChan, queryDoneChan chan struct{}
}

// NewRegistry returns a new registry instance.
func NewRegistry(dir, host, port string, queryInt time.Duration, logger *log.Logger) *registry {
	fs := serve.NewFileServer(serve.Options{
		Directory: dir,
		Prefix:    "/static/",
	})

	fs.Use(
		Logger(logger),
		Recover(),
		CORS(),
	)

	stmpl := template.Must(template.New("stream").Parse(streamPage))
	htmpl := template.Must(template.New("home").Parse(homePage))

	addr := net.JoinHostPort(host, port)
	mux := http.NewServeMux()
	r := &registry{
		srv: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			ErrorLog:     logger,
		},
		serviceMap:          make(map[string][]*mdns.ServiceEntry),
		resetServiceMapChan: make(chan struct{}),
		streamTmpl:          stmpl,
		homeTmpl:            htmpl,
		logger:              logger,
		queryInterval:       queryInt,
		entriesChan:         make(chan *mdns.ServiceEntry),
		queryDoneChan:       make(chan struct{}),
		quitChan:            make(chan struct{}),
	}

	mux.Handle("/static/", fs)
	mux.HandleFunc("/stream", r.streamHandler)
	mux.HandleFunc("/", r.homeHandler)

	return r
}

// Start the registry.
func (r *registry) Start() {
	go func() {
		// Start the server.
		err := r.srv.ListenAndServe()
		if err != http.ErrServerClosed {
			r.logger.Fatal(err)
		}
		// sending the quit signal.
		r.quitChan <- struct{}{}
	}()

	go r.readServiceEntries()

	r.lookupServices()

	// Start the query loop
	go r.queryLoop()
}

func (r *registry) readServiceEntries() {
	for {
		select {
		case <-r.resetServiceMapChan:
			r.servicesMu.Lock()
			r.serviceMap["stream_publisher._tcp"] = []*mdns.ServiceEntry{}
			r.serviceMap["proxy_cache._tcp"] = []*mdns.ServiceEntry{}
			r.servicesMu.Unlock()
		case entry := <-r.entriesChan:
			// Channel is closed, so exit
			if entry == nil {
				return
			}
			// read all the service entries.
			nameBits := strings.Split(entry.Name, ".")
			if len(nameBits) < 4 {
				r.logger.Printf("Service name: %q is incorrectly formatted\n")
				continue
			}
			name := strings.Join(nameBits[1:3], ".")
			r.servicesMu.Lock()
			r.serviceMap[name] = append(r.serviceMap[name], entry)
			r.servicesMu.Unlock()
		}
	}
}

// Stop the registry.
func (r *registry) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := r.srv.Shutdown(ctx)
	if err != nil {
		r.logger.Println(err)
	}
	cancel()

	// Wait for the http server to finish.
	<-r.quitChan
	r.logger.Println("Done with http server")

	// Exiting the query loop.
	r.queryDoneChan <- struct{}{}
	<-r.quitChan

	// Close the entries chan.
	r.logger.Println("Closing the entries chan")
	close(r.entriesChan)
}

func (r *registry) lookupServices() {
	// Start the lookup
	err := mdns.Lookup("stream_publisher._tcp", r.entriesChan)
	if err != nil {
		r.logger.Fatal(err)
	}

	err = mdns.Lookup("proxy_cache._tcp", r.entriesChan)
	if err != nil {
		r.logger.Fatal(err)
	}
}

func (r *registry) queryLoop() {
	ticker := time.NewTicker(r.queryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-r.queryDoneChan:
			r.logger.Println("Exiting from query loop")
			r.quitChan <- struct{}{}
			return
		case <-ticker.C:
			r.logger.Println("Refreshing service list")
			r.resetServiceMapChan <- struct{}{}
			r.lookupServices()
		}
	}
}
