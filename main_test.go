package pgx_test

import (
	"os"
	"testing"

	"github.com/goeventsource/pgx/pgxtest"
)

func TestMain(m *testing.M) {
	code := m.Run()
	pgxtest.CleanUp()
	os.Exit(code)
}
