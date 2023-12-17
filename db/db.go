// Copyright (C) 2021-2023 Nicola Murino
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, version 3.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package db

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/url"
	"os"
	"runtime"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
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
func Initialize(driver, dsn, customTLSConfig string, dbDebug bool) error {
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
		if err := handleCustomTLSConfig(customTLSConfig); err != nil {
			logger.AppLogger.Error("unable to register custom tls config", "error", err)
			return err
		}
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

func handleCustomTLSConfig(config string) error {
	if config == "" {
		return nil
	}
	values, err := url.ParseQuery(config)
	if err != nil {
		logger.AppLogger.Error("unable to parse custom tls config", "value", config, "error", err)
		return fmt.Errorf("unable to parse tls config: %w", err)
	}
	rootCert := values.Get("root_cert")
	clientCert := values.Get("client_cert")
	clientKey := values.Get("client_key")
	tlsMode := values.Get("tls_mode")

	tlsConfig := &tls.Config{}
	if rootCert != "" {
		rootCAs, err := x509.SystemCertPool()
		if err != nil {
			rootCAs = x509.NewCertPool()
		}
		rootCrt, err := os.ReadFile(rootCert)
		if err != nil {
			return fmt.Errorf("unable to load root certificate %q: %v", rootCert, err)
		}
		if !rootCAs.AppendCertsFromPEM(rootCrt) {
			return fmt.Errorf("unable to parse root certificate %q", rootCert)
		}
		tlsConfig.RootCAs = rootCAs
	}
	if clientCert != "" && clientKey != "" {
		cert := make([]tls.Certificate, 0, 1)
		tlsCert, err := tls.LoadX509KeyPair(clientCert, clientKey)
		if err != nil {
			return fmt.Errorf("unable to load key pair %q, %q: %v", clientCert, clientKey, err)
		}
		cert = append(cert, tlsCert)
		tlsConfig.Certificates = cert
	}
	if tlsMode == "1" {
		tlsConfig.InsecureSkipVerify = true
	}

	if err := mysqldriver.RegisterTLSConfig("custom", tlsConfig); err != nil {
		return fmt.Errorf("unable to register tls config: %v", err)
	}
	return nil
}
