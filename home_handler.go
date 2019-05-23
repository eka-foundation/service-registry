package main

import (
	"net"
	"net/http"
	"strings"
)

func (r *registry) homeHandler(w http.ResponseWriter, req *http.Request) {
	// The "/" pattern matches everything, so we need to check
	// that we're at the root here.
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	// We construct the data to be shown in the web page
	type item struct {
		OriginHost   net.IP
		OriginPort   int
		CacheHost    net.IP
		CachePort    int
		OriginName   string
		OriginPrefix string
		CacheName    string
	}
	ctx := []item{}
	r.servicesMu.RLock()
	// for every origin
	for _, oSvc := range r.serviceMap["stream_publisher._tcp"] {
		// XXX: (agnivade)- improve this logic later by sending the metadata
		// in 2 separate items
		streamMetadata := strings.Split(oSvc.Info, "_")
		if len(streamMetadata) < 2 {
			r.logger.Println("malformed stream publisher info: ", oSvc.Info)
			continue
		}
		originName := streamMetadata[0]
		prefix := streamMetadata[1]
		// for every cache
		for _, cSvc := range r.serviceMap["proxy_cache._tcp"] {
			ctx = append(ctx, item{
				OriginHost:   oSvc.AddrV4,
				OriginPort:   oSvc.Port,
				CacheHost:    cSvc.AddrV4,
				CachePort:    cSvc.Port,
				OriginName:   originName,
				OriginPrefix: prefix,
				CacheName:    cSvc.Info,
			})
		}
	}
	r.servicesMu.RUnlock()
	err := r.homeTmpl.Execute(w, ctx)
	if err != nil {
		r.logger.Println(err)
	}
}

const homePage = `<!DOCTYPE html>
<html>
<head>
<meta charset=utf-8 />
<title>Coordinator home page</title>
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<meta name="description" content="video livestream url">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
</head>
<body>
  <h1>Home page</h1>
  <p>
  <h3>Live streams available right now:</h3>
  {{range $item := .}}
    <a href="/stream?prefix={{.OriginPrefix}}&origin={{.OriginHost}}:{{.OriginPort}}&old=true">[oldjs-pointing to origin]From {{.OriginName}}, cached at {{.CacheName}}</a>
    <br />
    <a href="/stream?prefix={{.OriginPrefix}}&cache={{.CacheHost}}:{{.CachePort}}&old=true">[oldjs-pointing to cache]From {{.OriginName}}, cached at {{.CacheName}}</a>
    <br />
    <a href="/stream?prefix={{.OriginPrefix}}&origin={{.OriginHost}}:{{.OriginPort}}&old=false">[newjs-pointing to origin]From {{.OriginName}}, cached at {{.CacheName}}</a>
    <br />
    <a href="/stream?prefix={{.OriginPrefix}}&cache={{.CacheHost}}:{{.CachePort}}&old=false">[newjs-pointing to cache]From {{.OriginName}}, cached at {{.CacheName}}</a>
  {{end}}
  </p>
</body>
</html>
`
