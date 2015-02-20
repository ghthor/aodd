package game

import (
	"testing"

	"github.com/ghthor/gospec"
)

func TestUnitSpecs(t *testing.T) {
	r := gospec.NewRunner()

	r.AddSpec(DescribeActorConn)
	r.AddSpec(DescribeCollision)

	gospec.MainGoTest(r, t)
}
