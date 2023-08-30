# Autoshell

> [!NOTE]
> This README/documentation is incomplete.

Autoshell is a command-line utility facilitating automatic execution of shell commands under different environments.

[![Latest Release](https://img.shields.io/github/v/release/shibijm/autoshell?label=Latest%20Release)](https://github.com/shibijm/autoshell/releases/latest)
[![Build Status](https://img.shields.io/github/actions/workflow/status/shibijm/autoshell/release.yml?label=Build&logo=github)](https://github.com/shibijm/autoshell/actions/workflows/release.yml)

## Usage

```
Usage:
  autoshell [command]

Available Commands:
  config      Config file management
  run         Run a workflow

Flags:
  -c, --config string   config file path (default "config.yml")
  -h, --help            help for autoshell
  -v, --version         version for autoshell

Use "autoshell [command] --help" for more information about a command.
```

```
Config file management

Usage:
  autoshell config [command]

Available Commands:
  decrypt     Decrypt the config file
  encrypt     Encrypt the config file

Flags:
  -h, --help   help for config

Global Flags:
  -c, --config string   config file path (default "config.yml")

Use "autoshell config [command] --help" for more information about a command.
```

```
Run a workflow

Usage:
  autoshell run [workflow] [args] [flags]

Flags:
  -h, --help   help for run

Global Flags:
  -c, --config string   config file path (default "config.yml")
```

## Configuration - `config.yml`

### Encryption

Config file encryption employs AES-256 in GCM mode. The encryption key is derived from a provided password using Argon2id.

Instances of the string `$auto` within a given password will be substituted with a device pass. The 'run' command will attempt to automatically decrypt the config file using password `$auto` before prompting for manual password input. This operation proves successful when attempted on the original machine that initially encrypted the config file. This facilitates the obfuscation of config files while enabling fully automated runs without the need for manual password entry.

### Actions

- `runWorkflow [workflow] [args]`
- `setEnvVar [name] [value]`
- `setGlobalVar [name] [value]`
- `setLocalVar [name] [value]`
- `runCommand [commandID] [command] [args]`
- `setLogFile [path]`
- `addReporter uptimeKuma [url]`
- `setIgnoredErrorCodes [codes]`
- `print [args]`
- `shiftArgVars`

### Example

```yml
protected: true
workflows:
  main: |-
    setLogFile autoshell.log
    addReporter uptimeKuma https://example.com/api/push/oqQJiMo2DG
    runCommand backup-mysql mysqldump -u root -p4U5fUbmxtk myapp -r db.sql
    runWorkflow setup-restic
    runWorkflow setup-restic-storj
    runWorkflow restic-backup
    runWorkflow setup-restic-ext-hdd
    runWorkflow restic-backup
  restic-backup: |-
    runCommand $destination-backup-mysql $restic backup db.sql
    runCommand $destination-backup-code $restic backup D:\Code
    runCommand $destination-backup-documents $restic backup D:\Documents
  restic: |-
    runWorkflow setup-restic
    runWorkflow setup-restic-$1
    shiftArgVars
    runCommand $destination $restic $@
  setup-restic: |-
    setEnvVar RCLONE_CONFIG notfound
    setEnvVar RCLONE_BWLIMIT 8M
    setGlobalVar restic "restic --limit-download 8192 --limit-upload 8192"
    setIgnoredErrorCodes [3]
  setup-restic-storj: |-
    setGlobalVar destination storj
    setEnvVar RESTIC_REPOSITORY rclone::storj,access_grant=sA1PTsVFRR:restic
    setEnvVar RESTIC_PASSWORD dSeMFHqluz
  setup-restic-ext-hdd: |-
    setGlobalVar destination ext-hdd
    setEnvVar RESTIC_REPOSITORY F:\Restic
    setEnvVar RESTIC_PASSWORD 3seMRdKnS7
```

<details>

<summary>Sample runs</summary>

```
> autoshell run main
--------------------------------------------------------------------------------
Started at 2023-08-29T20:36:20.6646661+05:30
--------------------------------------------------------------------------------
Command ID: backup-mysql
--------------------------------------------------------------------------------
[command output]
--------------------------------------------------------------------------------
Command ID: storj-backup-mysql
--------------------------------------------------------------------------------
[command output]
--------------------------------------------------------------------------------
Command ID: storj-backup-code
--------------------------------------------------------------------------------
[command output]
--------------------------------------------------------------------------------
Command ID: storj-backup-documents
--------------------------------------------------------------------------------
[command output]
--------------------------------------------------------------------------------
Command ID: ext-hdd-backup-mysql
--------------------------------------------------------------------------------
[command output]
--------------------------------------------------------------------------------
Command ID: ext-hdd-backup-code
--------------------------------------------------------------------------------
[command output]
--------------------------------------------------------------------------------
Command ID: ext-hdd-backup-documents
--------------------------------------------------------------------------------
[command output]
--------------------------------------------------------------------------------
Ended at 2023-08-29T20:37:47.6888018+05:30 (took 87 seconds)
--------------------------------------------------------------------------------
```

```
> autoshell run restic ext-hdd snapshots -- --compact
--------------------------------------------------------------------------------
Started at 2023-08-29T20:43:15.7856331+05:30
--------------------------------------------------------------------------------
Command ID: ext-hdd
--------------------------------------------------------------------------------
repository 219fdcf6 opened (version 2, compression level auto)
ID        Time                 Host    Tags
---------------------------------------------
9476f199  2023-08-29 20:37:16  server
156ff78b  2023-08-29 20:37:28  server
41432ec6  2023-08-29 20:37:45  server
---------------------------------------------
3 snapshots
--------------------------------------------------------------------------------
Ended at 2023-08-29T20:43:20.5029457+05:30 (took 5 seconds)
--------------------------------------------------------------------------------
```

</details>
