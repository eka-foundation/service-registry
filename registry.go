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
	servicesMu           sync.RWMutex
	serviceMap           map[string][]*mdns.ServiceEntry
	srv                  *http.Server
	streamTmpl, homeTmpl *template.Template
	logger               *log.Logger
}

// NewRegistry returns a new registry instance.
func NewRegistry(dir, host, port string, logger *log.Logger) *registry {
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
		serviceMap: make(map[string][]*mdns.ServiceEntry),
		streamTmpl: stmpl,
		homeTmpl:   htmpl,
		logger:     logger,
	}

	mux.Handle("/static/", fs)
	mux.HandleFunc("/stream", r.streamHandler)
	mux.HandleFunc("/", r.homeHandler)

	return r
}

// Start the registry.
func (r *registry) Start() {
	entriesCh := make(chan *mdns.ServiceEntry)
	go func() {
		// read all the service entries.
		for entry := range entriesCh {
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
		// now start the server.
		err := r.srv.ListenAndServe()
		if err != http.ErrServerClosed {
			r.logger.Fatal(err)
		}
	}()

	// Start the lookup
	err := mdns.Lookup("stream_publisher._tcp", entriesCh)
	if err != nil {
		r.logger.Fatal(err)
	}

	err = mdns.Lookup("proxy_cache._tcp", entriesCh)
	if err != nil {
		r.logger.Fatal(err)
	}

	close(entriesCh)
}

// Stop the registry.
func (r *registry) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := r.srv.Shutdown(ctx)
	if err != nil {
		r.logger.Println(err)
	}
	cancel()
}
