package db

import (
	"strings"
	"testing"
)

func TestConnectRejectsInvalidDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "://invalid")

	pool, err := Connect()
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("expected invalid DATABASE_URL to fail")
	}
	if !strings.Contains(err.Error(), "unable to parse database config") {
		t.Fatalf("expected parse config error, got %v", err)
	}
}
