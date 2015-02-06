package www

import (
	"net/http"
	"testing"
	"text/template"

	"github.com/ghthor/aodd/game"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

func DescribeClient(c gospec.Context) {
	indexTmpl := template.Must(template.New("index.tmpl").ParseFiles("index.tmpl"))

	// TODO add a route to the http server that phantom can trigger
	// to signify that the tests are completed running
	shardConfig := game.ShardConfig{
		IsHTTPS: false,
		LAddr:   "localhost:45001",

		IndexTmpl: indexTmpl,

		JsDir:  "js/",
		JsMain: "js/main_test",

		Mux: http.NewServeMux(),
	}

	_, err := game.NewSimShard(shardConfig)
	c.Assume(err, IsNil)

	// TODO verify that phantomjs is installed.
	// It is the only external dependancy to run the client tests.
}

func TestClient(t *testing.T) {
	r := gospec.NewRunner()

	r.AddSpec(DescribeClient)

	gospec.MainGoTest(r, t)
}
