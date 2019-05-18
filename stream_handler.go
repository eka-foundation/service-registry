package main

import (
	"net/http"
)

func (r *registry) streamHandler(w http.ResponseWriter, req *http.Request) {
	// We construct the data to be shown in the web page
	type ctx struct {
		OriginAddr string
		CacheAddr  string
	}

	q := req.URL.Query()

	err := r.streamTmpl.Execute(w, ctx{
		OriginAddr: q.Get("origin"),
		CacheAddr:  q.Get("cache"),
	})
	if err != nil {
		r.logger.Println(err)
	}
}

const streamPage = `<!DOCTYPE html>
<html>
<head>
<meta charset=utf-8 />
<title>Video live stream</title>
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<meta name="description" content="video livestream url">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
  <link href="/static/css/video-js.css" rel="stylesheet">
</head>
<body>
  <h1>Video.js Example Embed</h1>

  <video id="player" class="video-js" controls preload="auto" width="640" height="268">
    <source src="//{{.CacheAddr}}/playlist.m3u8?origin={{.OriginAddr}}" type="application/x-mpegURL">
  </video>

  <script src="/static/js/video.js"></script>

  <script>
    var player = videojs('player', {
      autoplay: true,
      html5: {
        hlsjsConfig: {
          debug: true,
          // Other hlsjsConfig options provided by hls.js
          p2pConfig: {
            logLevel: true,
            live: false,        // set to true in live mode
          }
        }
      }
    });
  </script>
</body>
</html>
`
