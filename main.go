package main

import (
	"flag"
	"log"
	"net/http"
	"text/template"

	"github.com/ghthor/aodd/game"
)

var indexTmpl = template.Must(template.New("index.tmpl").ParseFiles("www/index.tmpl"))

func main() {
	laddrHTTP := flag.String("r", "localhost:8080",
		"address for a HTTP server that redirects to the HTTPS game server")
	laddrTLS := flag.String("s", "localhost:8081", "address for the HTTPS game server")

	certFile := flag.String("cert", "cert.pem", "TLS cert filepath")
	keyFile := flag.String("key", "key.pem", "TLS key filepath")

	flag.Parse()

	go func() {
		http.Handle("/", http.RedirectHandler("https://"+*laddrTLS, 301))

		log.Printf("started: redirect server http://%s -> https://%s", *laddrHTTP, *laddrTLS)
		err := http.ListenAndServe(*laddrHTTP, nil)

		if err != nil {
			log.Fatal("error in redirect server:", err)
		}
	}()

	c := game.ShardConfig{
		LAddr:   *laddrTLS,
		IsHTTPS: true,

		JsDir:    "www/js",
		AssetDir: "www/asset",
		CssDir:   "www/css",

		JsMain: "js/init",

		IndexTmpl: indexTmpl,

		Mux: http.NewServeMux(),
	}

	s, err := game.NewSimShard(c)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("started: tls server https://%s", *laddrTLS)
	err = s.ListenAndServeTLS(*certFile, *keyFile)
	if err != nil {
		log.Fatal(err)
	}
}
