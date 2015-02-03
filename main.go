package main

import (
	"flag"
	"log"
	"net/http"
)

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

	s, err := newSimShard(*laddrTLS)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("started: tls server https://%s", *laddrTLS)
	err = s.ListenAndServeTLS(*certFile, *keyFile)
	if err != nil {
		log.Fatal(err)
	}
}
