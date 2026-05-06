package di

import (
	"io"
	"log"
	"testing"
)

func testLogger(t *testing.T) *log.Logger {
	t.Helper()
	return log.New(io.Discard, "[test] ", 0)
}
