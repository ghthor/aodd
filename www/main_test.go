package www

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"text/template"
	"time"

	"github.com/ghthor/aodd/game"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

var browser string

// Used to store executable paths
var phantomjs string
var chromium string

func init() {
	flag.StringVar(&browser, "browser", "phantomjs", "the browser engine used to run the specifications")
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

// Starts an http game server and verifies that is
// can respond to http requests before returning it.
func startWebServer(shardConfig game.ShardConfig) (*http.Server, error) {
	hasStarted := make(chan struct{}, 1)

	// Set a route that can be used
	// to trigger starting the webserver
	shardConfig.Mux.Handle("/start", triggerStartHandler{hasStarted})

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

func DescribeConsoleReport(c gospec.Context) {
	indexTmpl := template.Must(template.New("index.tmpl").ParseFiles("index.tmpl"))

	domain := "localhost"
	port := "45001"
	laddr := fmt.Sprintf("%s:%s", domain, port)

	shardConfig := game.ShardConfig{
		Domain: domain,
		Port:   port,

		JsDir:    "js/",
		AssetDir: "asset/",
		CssDir:   "css/",

		JsMain: "js/specs_console_report",

		IndexTmpl: indexTmpl,

		Mux: http.NewServeMux(),
	}

	_, err := startWebServer(shardConfig)
	if err != nil {
		// Print out error and exit early
		c.Assume(err, IsNil)
		return
	}

	phantomjsScriptTmpl := template.Must(template.New("phantomjs_specs.js.tmpl").ParseFiles("phantomjs_specs.js.tmpl"))

	// Create a file in os temp directory
	tmpFile, err := ioutil.TempFile("", "phantomjs_specs.js")
	c.Assume(err, IsNil)

	// Remove temp file
	defer func() {
		c.Assume(os.RemoveAll(tmpFile.Name()), IsNil)
	}()

	type phantomjsTemplate struct {
		LAddr string
	}

	// Write out the file using the script template
	c.Assume(phantomjsScriptTmpl.Execute(tmpFile, phantomjsTemplate{laddr}), IsNil)

	cmd := exec.Command(phantomjs, tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	// Run jasmine specs through phantomjs
	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	err = cmd.Wait()
	c.Assume(err, IsNil)
}

type specsHaveBeenRanHandler struct {
	haveCompleted chan<- struct{}
}

func (specs specsHaveBeenRanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "server shutting down")
	specs.haveCompleted <- struct{}{}
}

func DescribeHtmlReport(c gospec.Context) {
	indexTmpl := template.Must(template.New("index.tmpl").ParseFiles("index.tmpl"))

	domain := "localhost"
	port := "45001"
	laddr := fmt.Sprintf("%s:%s", domain, port)

	specsHaveBeenRan := make(chan struct{})

	mux := http.NewServeMux()
	mux.Handle("/specs/complete", specsHaveBeenRanHandler{specsHaveBeenRan})

	shardConfig := game.ShardConfig{
		Domain: domain,
		Port:   port,

		JsDir:    "js/",
		AssetDir: "asset/",
		CssDir:   "css/",

		JsMain: "js/specs_html_report",

		IndexTmpl: indexTmpl,

		Mux: mux,
	}

	_, err := startWebServer(shardConfig)
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

	// Wait for tests to signal
	// they have been run.
	<-specsHaveBeenRan
}

func TestRunJasmineSpecs(t *testing.T) {
	var err error

	r := gospec.NewRunner()

	switch browser {
	case "phantomjs":
		phantomjs, err = exec.LookPath("phantomjs")
		if err != nil {
			t.Fatal("phantomjs must be installed")
		}

		r.AddSpec(DescribeConsoleReport)

	case "chromium":
		chromium, err = exec.LookPath("chromium")
		if err != nil {
			t.Fatal("chromium must be installed")
		}

		r.AddSpec(DescribeHtmlReport)

	default:
		t.Fatal(browser, "is unimplemented as an engine target")
	}

	gospec.MainGoTest(r, t)
}
