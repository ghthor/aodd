package main

import (
	"testing"

	"github.com/ghthor/gospec"
)

func TestUnitSpecs(t *testing.T) {
	r := gospec.NewRunner()

	r.AddSpec(DescribeActorConn)

	gospec.MainGoTest(r, t)
}
