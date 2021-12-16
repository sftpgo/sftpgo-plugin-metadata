package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/sftpgo/sftpgo-plugin-metadata/logger"
)

const (
	driverNamePostgreSQL = "postgres"
	driverNameMySQL      = "mysql"
	cleanupQuery         = `DELETE FROM metadata_folders WHERE NOT EXISTS
 (SELECT id FROM metadata_files WHERE metadata_files.folder_id = metadata_folders.id)`
)

var (
	Handle              *gorm.DB
	defaultQueryTimeout = 20 * time.Second
)

// Initialize initializes the database engine
func Initialize(driver, dsn string, dbDebug bool) error {
	var err error

	newLogger := gormlogger.Discard

	if dbDebug {
		newLogger = gormlogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			gormlogger.Config{
				SlowThreshold: time.Second,     // Slow SQL threshold
				LogLevel:      gormlogger.Info, // Log level
				Colorful:      runtime.GOOS != "windows",
			},
		)
	}

	switch driver {
	case driverNamePostgreSQL:
		Handle, err = gorm.Open(postgres.New(postgres.Config{
			DSN: dsn,
		}), &gorm.Config{
			SkipDefaultTransaction: true,
			Logger:                 newLogger,
		})
		if err != nil {
			logger.AppLogger.Error("unable to create db handle", "error", err)
			return err
		}
	case driverNameMySQL:
		Handle, err = gorm.Open(mysql.New(mysql.Config{
			DSN: dsn,
		}), &gorm.Config{
			SkipDefaultTransaction: true,
			Logger:                 newLogger,
		})
		if err != nil {
			logger.AppLogger.Error("unable to create db handle", "error", err)
			return err
		}
	default:
		return fmt.Errorf("unsupported database driver %v", driver)
	}

	sqlDB, err := Handle.DB()
	if err != nil {
		logger.AppLogger.Error("unable to get sql db handle", "error", err)
		return err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxIdleTime(3 * time.Minute)

	return sqlDB.Ping()
}

// ScheduleCleanup tries to periodically remove unreferenced folders
func ScheduleCleanup() {
	for range time.Tick(12 * time.Hour) {
		logger.AppLogger.Debug("removing unreferenced folders")
		err := removeUnreferencedFolders()
		logger.AppLogger.Info("removing unreferenced folders completed", "error", err)
	}
}

func removeUnreferencedFolders() error {
	sess, cancel := getSessionWithTimeout(defaultQueryTimeout * 4)
	defer cancel()

	return sess.Exec(cleanupQuery).Error
}

// getDefaultSession returns a database session with the default timeout.
// Don't forget to cancel the returned context
func getDefaultSession() (*gorm.DB, context.CancelFunc) {
	return getSessionWithTimeout(defaultQueryTimeout)
}

// getSessionWithTimeout returns a database session with the specified timeout.
// Don't forget to cancel the returned context
func getSessionWithTimeout(timeout time.Duration) (*gorm.DB, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	return Handle.WithContext(ctx), cancel
}

// executeTx runs txFn inside a transaction
func executeTx(db *gorm.DB, txFn func(tx *gorm.DB) error) error {
	err := db.Transaction(txFn)
	if err != nil {
		logger.AppLogger.Error("unable to execute transaction", "error", err)
	}
	return err
}

func checkRowsAffected(tx *gorm.DB) error {
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
