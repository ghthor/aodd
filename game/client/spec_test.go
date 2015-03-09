package client_test

import (
	"testing"

	"github.com/ghthor/aodd/game/prototest"
	"github.com/ghthor/gospec"
)

func TestUnitSpecs(t *testing.T) {
	r := gospec.NewRunner()

	r.AddSpec(prototest.DescribeActorGobConn)

	gospec.MainGoTest(r, t)
}
