package main

import "testing"

func TestLoopTestsSkipped(t *testing.T) {
	t.Skip("loop tests are skipped; refactor handlers to accept context/ticker for reliable testing")
}
