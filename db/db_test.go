package db

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sftpgo/sftpgo-plugin-metadata/db/migration"
)

func TestMain(m *testing.M) {
	driver := os.Getenv("SFTPGO_PLUGIN_METADATA_DRIVER")
	dsn := os.Getenv("SFTPGO_PLUGIN_METADATA_DSN")
	if driver == "" || dsn == "" {
		fmt.Println("Driver and/or DSN not set, unable to execute test")
		os.Exit(1)
	}
	if err := Initialize(driver, dsn, true); err != nil {
		fmt.Printf("unable to initialize database: %v\n", err)
		os.Exit(1)
	}
	if err := migration.MigrateDatabase(Handle); err != nil {
		fmt.Printf("unable to migrate database: %v\n", err)
		os.Exit(1)
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}

func getTimeAsMsSinceEpoch(t time.Time) int64 {
	return t.UnixNano() / 1000000
}
