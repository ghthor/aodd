package client_test

import (
	"testing"

	"github.com/ghthor/gospec"
)

func TestUnitSpecs(t *testing.T) {
	r := gospec.NewRunner()

	// TODO This test is disabled because of the change to startIO
	//      that removes the sending of the world diffs from within
	//      the IO loop.
	//r.AddSpec(prototest.DescribeActorGobConn)

	gospec.MainGoTest(r, t)
}
