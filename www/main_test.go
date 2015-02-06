package www

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"text/template"

	"github.com/ghthor/aodd/game"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

type hasStartedHandler struct {
	hasStarted chan<- struct{}
}

// TODO This could accept POST test data that could be
// checked and displayed here instead of phantomjs's stdout.
func (s hasStartedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.hasStarted <- struct{}{}
	fmt.Fprint(w, "has started")
}

var phantomjs string

func init() {
	var err error
	phantomjs, err = exec.LookPath("phantomjs")
	if err != nil {
		fmt.Println("phantomjs must be installed")
		os.Exit(1)
	}
}

func DescribeClient(c gospec.Context) {
	indexTmpl := template.Must(template.New("index.tmpl").ParseFiles("index.tmpl"))

	hasStarted := make(chan struct{})

	mux := http.NewServeMux()
	mux.Handle("/triggerStart", hasStartedHandler{hasStarted})

	shardConfig := game.ShardConfig{
		IsHTTPS: false,
		LAddr:   "localhost:45001",

		IndexTmpl: indexTmpl,

		JsDir:  "js/",
		JsMain: "js/main_test",

		AssetDir: "img/",

		Mux: mux,
	}

	s, err := game.NewSimShard(shardConfig)
	c.Assume(err, IsNil)

	// Start the http server
	go func() {
		c.Assume(s.ListenAndServe(), IsNil)
	}()

	// Trigger s.ListAndServe()
	go func() {
		_, err := http.Get("http://localhost:45001/triggerStart")
		c.Assume(err, IsNil)
	}()

	// Wait for conformation that the server is live and listening
	<-hasStarted

	cmd := exec.Command(phantomjs, "client_test.js")
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

	//<-testsHaveCompleted
}

func TestClient(t *testing.T) {
	r := gospec.NewRunner()

	r.AddSpec(DescribeClient)

	gospec.MainGoTest(r, t)
}
