package game_test

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"text/template"
	"time"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/aodd/game/prototest"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

var web bool

var chromium string

func init() {
	flag.BoolVar(&web, "web", false, "start a web server and launch a browser to test in")
	flag.Parse()
}

type triggerStartHandler struct {
	hasStarted chan<- struct{}
}

// TODO This could accept POST test data that could be
// checked and displayed here instead of phantomjs's stdout.
func (s triggerStartHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.hasStarted <- struct{}{}
	fmt.Fprint(w, "server is running")
}

type disableCache struct {
	*http.ServeMux
}

func (m disableCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "no-cache")
	m.ServeMux.ServeHTTP(w, r)
}

// Starts an http game server and verifies that is
// can respond to http requests before returning it.
func startWebServer(shardConfig game.ShardConfig) (*http.Server, error) {
	hasStarted := make(chan struct{}, 1)

	// Set a route that can be used
	// to trigger starting the webserver
	shardConfig.Mux.Handle("/start", triggerStartHandler{hasStarted})
	shardConfig.Handler = disableCache{shardConfig.Mux}

	s, err := game.NewSimShard(shardConfig)
	if err != nil {
		return nil, err
	}

	// Used to collect errors from go routines that are
	// forcing the server to be initialized and started.
	errch := make(chan error, 20)

	// Start the http server
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			errch <- err
		}
	}()

	// Trigger s.ListenAndServe()
	go func() {
		// TODO Build this url using the shard config
		_, err := http.Get("http://localhost:45001/start")
		if err != nil {
			errch <- err
		}
	}()

	// Set a timeout incase the server borks itself
	ticker := time.NewTicker(time.Second * 1)

	// Verify that the server has been started
	select {
	case <-ticker.C:
		errch <- errors.New("timeout waiting for http server to be started")
	case <-hasStarted:
	}

	// See if there were any errors during initialization
	if len(errch) > 0 {
		for e := range errch {
			if e != nil {
				return nil, e
			}
		}
	}

	return s, nil
}

type completedTriggerHandler struct {
	hasCompleted chan<- struct{}
}

func (testing completedTriggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "server shutting down")
	testing.hasCompleted <- struct{}{}
}

func DescribeLiveWebTesting(c gospec.Context) {
	indexTmpl := template.Must(template.New("index.tmpl").ParseFiles("../www/index.tmpl"))

	port := "45001"
	laddr := fmt.Sprintf("localhost:%s", port)

	userClosedBrowser := make(chan struct{})

	mux := http.NewServeMux()
	mux.Handle("/testing/complete", completedTriggerHandler{userClosedBrowser})

	var err error

	f, err := os.OpenFile("login_test.js", os.O_RDONLY, 0666)
	if err != nil {
		c.Assume(err, IsNil)
		return
	}
	defer f.Close()

	mux.HandleFunc("/js/ui/login.js", func(w http.ResponseWriter, r *http.Request) {
		io.Copy(w, f)
		f.Seek(0, 0)
	})

	i := 0

	mux.HandleFunc("/actor/unique", func(w http.ResponseWriter, r *http.Request) {
		defer func() { i++ }()
		msg := struct {
			Id string `json:"id"`
		}{
			fmt.Sprint(i),
		}

		bytes, err := json.Marshal(msg)
		if err != nil {
			panic(err)
		}

		fmt.Fprint(w, string(bytes))
	})

	shardConfig := game.ShardConfig{
		LAddr:   laddr,
		IsHTTPS: false,

		JsDir:    "../www/js/",
		AssetDir: "../www/asset/",
		CssDir:   "../www/css/",

		JsMain: "js/init",

		IndexTmpl: indexTmpl,

		Mux: mux,
	}

	_, err = startWebServer(shardConfig)
	if err != nil {
		// Print out error and exit early
		c.Assume(err, IsNil)
		return
	}

	err = exec.Command(chromium, "--incognito", "http://"+laddr).Run()
	if err != nil {
		c.Assume(err, IsNil)
		return
	}

	// Wait for user to signal they're done
	<-userClosedBrowser
}

func TestUnitSpecs(t *testing.T) {
	r := gospec.NewRunner()

	r.AddSpec(DescribeGobConn)
	r.AddSpec(prototest.DescribeActorGobConn)

	r.AddSpec(game.DescribeActorState)

	r.AddSpec(game.Describe2Actors)
	r.AddSpec(game.Describe3Actors)
	r.AddSpec(game.DescribeSomeActors)

	var err error

	if web {
		chromium, err = exec.LookPath("chromium")
		if err != nil {
			t.Fatal("chromium must be installed")
		}

		r.AddSpec(DescribeLiveWebTesting)
	}

	gospec.MainGoTest(r, t)
}
