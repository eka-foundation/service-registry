package main

import (
	"html/template"
	"net/http"
)

func (r *registry) streamHandler(w http.ResponseWriter, req *http.Request) {
	// We construct the data to be shown in the web page
	type ctx struct {
		OriginPrefix string
		CacheAddr    string
		TargetAddr   string // maybe origin or cache. decided dynamically.
	}

	q := req.URL.Query()

	targetAddr := ""
	if q.Get("cache") != "" {
		targetAddr = q.Get("cache")
	} else if q.Get("origin") != "" {
		targetAddr = q.Get("origin")
	} else {
		http.Error(w, "neither cache nor origin query param is set", http.StatusBadRequest)
		return
	}

	var tmpl *template.Template
	isOldJS := q.Get("old")
	if isOldJS == "" {
		http.Error(w, "old query param not set", http.StatusBadRequest)
		return
	}
	if isOldJS == "true" {
		tmpl = template.Must(template.New("stream").Parse(streamPageOld))
	} else if isOldJS == "false" {
		tmpl = template.Must(template.New("stream").Parse(streamPage))
	} else {
		http.Error(w, "old query param is something other than true or false", http.StatusBadRequest)
		return
	}

	// TODO: bring back cached parsed template once issue is resolved.
	// For now, we are dynamically parsing the template depending on the query param.
	err := tmpl.Execute(w, ctx{
		OriginPrefix: q.Get("prefix"),
		TargetAddr:   targetAddr,
	})
	if err != nil {
		r.logger.Println(err)
	}
}

const streamPageOld = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>CDNBye videojs5 demo</title>
</head>
<link href="/static/css/video-js.min.css" rel="stylesheet">
<script src="/static/js/video.min.js"></script>
<!-- increase browser support with MSE polyfill -->
<script src="/static/js/videojs-contrib-media-sources.min.js"></script>
<script src="/static/js/cdnbye.js"></script>
<script src="/static/js/videojs-contrib-hlsjs.min.js"></script>

<body>
<div id="main">
    <video id="player" class="video-js vjs-default-skin" height="360" width="640" controls preload="none">
        <source src="//{{.TargetAddr}}/{{.OriginPrefix}}_playlist.m3u8" type="application/x-mpegURL"/>
    </video>
    <p id="version"></p>
    <h3>download info:</h3>
    <p id="info"></p>
</div>
<script>
    var player = videojs('#player', {
        autoplay: true,
        html5: {
            hlsjsConfig: {
                debug: true,
                // Other hlsjsConfig options provided by hls.js
                p2pConfig: {
                    logLevel: true,
                    live: false        // set to true in live mode
                }
            }
        }
    });
</script>
</body>
</html>
`

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
  </video>
  <script src="/static/js/video.js"></script>
  <script>
    var player = videojs('player', {
      autoplay: true,
      html5: {
        nativeAudioTracks: false,
        nativeVideoTracks: false,
        hls: {
          overrideNative: true
        }
      }
    });
    player.src({
      src: '//{{.TargetAddr}}/{{.OriginPrefix}}_playlist.m3u8',
      type: 'application/x-mpegURL'
    })
  </script>
</body>
</html>
`
