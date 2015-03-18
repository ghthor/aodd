package canvas_test

import (
	"testing"

	"github.com/ghthor/gospec"
)

func TestUnitSpecs(t *testing.T) {
	r := gospec.NewRunner()

	r.AddSpec(DescribeTerrainCanvas)

	gospec.MainGoTest(r, t)
}
