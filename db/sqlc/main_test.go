package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/bgebrechristos/simplebank/util"
	_ "github.com/lib/pq"
)

var testQueries *Queries
var testDB *sql.DB

// TestMain is the main entry point that all tests use
func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../..")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	testDB, err = sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testQueries = New(testDB)

	os.Exit(m.Run()) // Report it back to test runner
}
