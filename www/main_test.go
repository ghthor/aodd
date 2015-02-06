package www

import (
	"testing"
	"text/template"

	"github.com/ghthor/aodd/game"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

func DescribeClient(c gospec.Context) {
	indexTmpl := template.Must(template.New("index.tmpl").ParseFiles("index.tmpl"))

	_, err := game.NewSimShard("localhost:45001", indexTmpl, "js/main_test")
	c.Assume(err, IsNil)
}

func TestClient(t *testing.T) {
	r := gospec.NewRunner()

	r.AddSpec(DescribeClient)

	gospec.MainGoTest(r, t)
}
