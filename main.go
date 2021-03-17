package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"text/template"

	_ "net/http/pprof"

	"github.com/ghthor/aodd/game"
)

var indexTmpl = template.Must(template.New("index.tmpl").ParseFiles("www/index.tmpl"))

func serverUrl(onHeroku bool, domain, port string) string {
	url := "http://" + domain
	if !onHeroku {
		url += ":" + port
	}

	return url
}

func main() {
	domain := os.Getenv("DOMAIN")
	port := os.Getenv("PORT")

	isHeroku := flag.Bool("heroku", true, "enable is the app is running on heroku")
	flag.Parse()

	c := game.ShardConfig{
		OnHeroku: *isHeroku,

		Domain: domain,
		Port:   port,

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

	go func() {
		log.Println("starting profiling server at", "http://localhost:6060")
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("starting server at", serverUrl(*isHeroku, domain, port))
	err = s.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
