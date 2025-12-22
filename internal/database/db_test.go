package database_test

import (
	"database/sql"
	"testing"

	"github.com/markcromwell/gator/internal/database"
)

func TestNewAndWithTx(t *testing.T) {
	q := database.New(nil)
	if q == nil {
		t.Fatalf("expected non-nil Queries from New")
	}

	// WithTx should return a Queries instance even if tx is nil
	var tx *sql.Tx = nil
	q2 := q.WithTx(tx)
	if q2 == nil {
		t.Fatalf("expected non-nil Queries from WithTx")
	}
}
