# SFTPGo metadata plugin

![Build](https://github.com/sftpgo/sftpgo-plugin-metadata/workflows/Build/badge.svg?branch=main&event=push)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPLv3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

This plugin allows to store file metadata in supported database engines.

## Supported metadata

The plugin supports the following metadata:

- `modification time`, it allows to support changing modification times for cloud storage backends (S3, Azure blob, Google Cloud Storage). So you can preserve modification times even when uploading files to cloud storage backends. Cloud storage backends have a flat structure instead of a hierarchy like you would see in a file system. Folders are just the prefix of the files. The plugin does not support setting "folder" modification time.

## Configuration

The plugin can be configured within the `plugins` section of the SFTPGo configuration file. To start the plugin you have to use the `serve` subcommand. Here is the usage.

```shell
NAME:
   sftpgo-plugin-metadata serve - Launch the SFTPGo plugin, it must be called from an SFTPGo instance

USAGE:
   sftpgo-plugin-metadata serve [command options] [arguments...]

OPTIONS:
   --driver value  Database driver (required) [$SFTPGO_PLUGIN_METADATA_DRIVER]
   --dsn value     Data source URI (required) [$SFTPGO_PLUGIN_METADATA_DSN]
   --help, -h      show help (default: false)
```

The `driver` and `dsn` flags are required. Each flag can also be set using environment variables, for example the DSN can be set using the `SFTPGO_PLUGIN_METADATA_DSN` environment variable.

This is an example configuration.

```json
...
  "plugins": [
    {
      "type": "metadata",
      "cmd": "<path to sftpgo-plugin-metadata>",
      "args": ["serve", "--driver", "postgres"],
      "sha256sum": "",
      "auto_mtls": true
    }
  ]
...
```

With the above example the plugin is configured to connect to PostgreSQL. We set the DSN using the `SFTPGO_PLUGIN_METADATA_DSN` environment variable.

The plugin will not start if it fails to connect to the configured database service, this will prevent SFTPGo from starting.

The plugin supports also the `migrate` and `reset` sub-commands that can be used in standalone mode and are useful for debugging purposes. Please refer to their help texts for usage.

## Database tables

The plugin will automatically create the following database tables:

- `metadata_folders`
- `metadata_files`

Inspect your database for more details.

## Supported database services

### PostgreSQL

To use Postgres you have to use `postgres` as driver. If you have a database named `sftpgo_metadata` on localhost and you want to connect to it using the user `sftpgo` with the password `sftpgopass` you can use a DSN like the following one.

```shell
"host='127.0.0.1' port=5432 dbname='sftpgo_metadata' user='sftpgo' password='sftpgopass' sslmode=disable connect_timeout=10"
```

Please refer to the documentation [here](https://github.com/go-gorm/postgres) for details about the dsn.

### MySQL

To use MySQL you have to use `mysql` as driver. If you have a database named `sftpgo_metadata` on localhost and you want to connect to it using the user `sftpgo` with the password `sftpgopass` you can use a DSN like the following one.

```shell
"sftpgo:sftpgopass@tcp([127.0.0.1]:3306)/sftpgo_metadata?charset=utf8mb4&interpolateParams=true&timeout=10s&tls=false&writeTimeout=10s&readTimeout=10s&parseTime=true"
```

Please refer to the documentation [here](https://github.com/go-gorm/mysql) for details about the dsn.
