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

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-plugin"
	"github.com/sftpgo/sdk/plugin/metadata"
	"github.com/urfave/cli/v2"

	"github.com/sftpgo/sftpgo-plugin-metadata/db"
	"github.com/sftpgo/sftpgo-plugin-metadata/db/migration"
	"github.com/sftpgo/sftpgo-plugin-metadata/logger"
)

const (
	version   = "1.0.11"
	envPrefix = "SFTPGO_PLUGIN_METADATA_"
)

var (
	commitHash = ""
	buildDate  = ""
)

var (
	driver          string
	dsn             string
	customTLSConfig string

	dbFlags = []cli.Flag{
		&cli.StringFlag{
			Name:        "driver",
			Usage:       "Database driver (required)",
			Destination: &driver,
			EnvVars:     []string{envPrefix + "DRIVER"},
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "dsn",
			Usage:       "Data source URI (required)",
			Destination: &dsn,
			EnvVars:     []string{envPrefix + "DSN"},
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "custom-tls",
			Usage:       "Custom TLS config for MySQL driver (optional)",
			Destination: &customTLSConfig,
			EnvVars:     []string{envPrefix + "CUSTOM_TLS"},
			Required:    false,
		},
	}

	rootCmd = &cli.App{
		Name:    "sftpgo-plugin-metadata",
		Version: getVersionString(),
		Usage:   "SFTPGo metadata plugin",
		Commands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Launch the SFTPGo plugin, it must be called from an SFTPGo instance",
				Flags: dbFlags,
				Action: func(c *cli.Context) error {
					logger.AppLogger.Info("starting sftpgo-plugin-metadata", "version", getVersionString(),
						"database driver", driver)
					if err := db.Initialize(driver, dsn, customTLSConfig, false); err != nil {
						logger.AppLogger.Error("unable to initialize database", "error", err)
						return err
					}
					if err := migration.MigrateDatabase(db.Handle); err != nil {
						logger.AppLogger.Error("unable to migrate database", "error", err)
						return err
					}

					go db.ScheduleCleanup()

					plugin.Serve(&plugin.ServeConfig{
						HandshakeConfig: metadata.Handshake,
						Plugins: map[string]plugin.Plugin{
							metadata.PluginName: &metadata.Plugin{Impl: &db.Metadater{}},
						},
						GRPCServer: plugin.DefaultGRPCServer,
					})

					return errors.New("the plugin exited unexpectedly")
				},
			},
			{
				Name:  "migrate",
				Usage: "Apply database schema migrations",
				Flags: dbFlags,
				Action: func(c *cli.Context) error {
					if err := db.Initialize(driver, dsn, customTLSConfig, true); err != nil {
						logger.AppLogger.Error("unable to initialize database", "error", err)
						return err
					}
					if err := migration.MigrateDatabase(db.Handle); err != nil {
						logger.AppLogger.Error("unable to migrate database", "error", err)
						return err
					}
					return nil
				},
			},
			{
				Name:  "reset",
				Usage: "Reset the database schema, any data will be lost",
				Flags: dbFlags,
				Action: func(c *cli.Context) error {
					fmt.Println("You are about to delete all database data and schema", "driver", fmt.Sprintf("%#v", driver),
						"dsn", fmt.Sprintf("%#v", dsn), "Are you sure?")
					fmt.Println("Y/n")
					reader := bufio.NewReader(os.Stdin)
					answer, err := reader.ReadString('\n')
					if err != nil {
						fmt.Println("unexpected error", err)
						return err
					}
					if strings.ToUpper(strings.TrimSpace(answer)) != "Y" {
						fmt.Println("Aborted!")
						return errors.New("command aborted")
					}
					if err := db.Initialize(driver, dsn, customTLSConfig, true); err != nil {
						logger.AppLogger.Error("unable to initialize database", "error", err)
						return err
					}
					if err := migration.ResetDatabase(db.Handle); err != nil {
						logger.AppLogger.Error("unable to reset database", "error", err)
						return err
					}
					return nil
				},
			},
		},
	}
)

// Execute runs the root command
func Execute() error {
	return rootCmd.Run(os.Args)
}

func getVersionString() string {
	var sb strings.Builder
	sb.WriteString(version)
	if commitHash != "" {
		sb.WriteString("-")
		sb.WriteString(commitHash)
	}
	if buildDate != "" {
		sb.WriteString("-")
		sb.WriteString(buildDate)
	}
	return sb.String()
}
