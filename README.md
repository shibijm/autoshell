# Autoshell

Autoshell is a command-line utility facilitating automatic execution of shell commands.

[![Latest Release](https://img.shields.io/github/v/release/shibijm/autoshell?label=Latest%20Release)](https://github.com/shibijm/autoshell/releases/latest)
[![Build Status](https://img.shields.io/github/actions/workflow/status/shibijm/autoshell/release.yml?label=Build&logo=github)](https://github.com/shibijm/autoshell/actions/workflows/release.yml)

## Download

Downloadable builds are available on the [releases page](https://github.com/shibijm/autoshell/releases).

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

Config file encryption employs AES-256 in GCM mode. The encryption key is derived using Argon2id.

Instances of the string `$auto` within passwords will be substituted with a device pass, which is the SHA-256 hash of the combination of machine ID, hardcoded randomness and a random config file ID generated afresh each time before encrypting a config file.

The `run` command will attempt to automatically decrypt the config file using password `$auto` before prompting for manual password input. Such an attempt would be successful only on the machine that initially encrypted the config file. Config files used for fully automated runs can be obfuscated by using this feature.

If a config file is marked as protected, the `decrypt` command will refuse to save the decrypted data to disk if the decryption password contains `$auto`. In such cases, the underlying explicit password is required, which is displayed only once right after encryption.

<details>

<summary>Example</summary>

<br />

```
$ cat config.yml
protected: true
workflows:
  hello: runCommand - echo Hello world

$ autoshell config encrypt
Password: $auto (hidden input)
Confirm Password: $auto (hidden input)
Encryption password contains "$auto"
Config file is marked as protected and hence cannot be saved after decryption if the decryption password contains "$auto"
Please store this explicit password safely: 7887c80ba98ffce76d5650bce0cb56b12b56d81f0354201a7f9f67b194ebc019
Config file encrypted successfully

$ cat config.yml
o□□□□2o□r□k{□?;□□<□lKO□□□h□,□□{z□       ULX1
□□v[□s3□&\□□R□y                             □□□"□□□n□□[e□□□H□□:Xs□□□□|□□j□z□□□rY□□>□c□□2□<U□H□□□(BO□Պ□□2v~□□\f□□□□ꕦ□□

$ autoshell run hello
--------------------------------------------------------------------------------
Started at 2025-05-11T19:18:13.1756292+05:30
--------------------------------------------------------------------------------
Hello world
--------------------------------------------------------------------------------
Ended at 2025-05-11T19:18:13.2735076+05:30 after 97ms
--------------------------------------------------------------------------------

$ autoshell config decrypt
Password: $auto (hidden input)
Error: Config file is marked as protected, refusing to save the decrypted data to disk since the decryption password contains "$auto"

$ autoshell config decrypt
Password: 7887c80ba98ffce76d5650bce0cb56b12b56d81f0354201a7f9f67b194ebc019 (hidden input)
Config file decrypted successfully

$ cat config.yml
protected: true
workflows:
  hello: runCommand - echo Hello world
```

</details>

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

Append `!W` or `!L` to action names to restrict their execution to Windows or Linux respectively.

### Variable Substitution

`$x` would get substituted with the value of variable `x`.

### Example

```yml
protected: true
workflows:
  main: |-
    setLogFile autoshell.log
    addReporter uptimeKuma https://yourdomain/api/push/oqQJiMo2DG
    runCommand create-sql-dump mysqldump -u root -p4U5fUbmxtk myapp -r db.sql
    runWorkflow setup-restic b2
    runWorkflow restic-backup
    runWorkflow setup-restic ext-hdd
    runWorkflow restic-backup
    runCommand delete-sql-dump rm db.sql
  restic-backup: |-
    runCommand $resticDestination-backup-db $restic backup db.sql
    runCommand $resticDestination-backup-code $restic backup D:\Code
    runCommand $resticDestination-backup-documents $restic backup D:\Documents
  restic: |-
    runWorkflow setup-restic
    shiftArgVars
    runCommand restic-$resticDestination $restic $@
  setup-restic: |-
    setEnvVar RCLONE_CONFIG notfound
    setEnvVar RCLONE_FAST_LIST true
    setEnvVar RCLONE_BWLIMIT 8M
    setIgnoredErrorCodes [3]
    setGlobalVar resticDestination $1
    runWorkflow setup-restic-$resticDestination
    setGlobalVar restic "restic --limit-download 8192 --limit-upload 8192"
  setup-restic-b2: |-
    setEnvVar RCLONE_B2_ACCOUNT 2No2MBrvcnNzV4U4rQs2rq27h
    setEnvVar RCLONE_B2_KEY 7P7TWhWLK56P53zTRqZQw2aFriJcD6X
    setEnvVar RESTIC_REPOSITORY rclone::b2:restic
    setEnvVar RESTIC_PASSWORD k3Tw883j8QqMdDyG2TPt6jfo9iZR9hu7M4Zo43zE7vYf3brDjtkAhxF3T9DoHkjj
  setup-restic-ext-hdd: |-
    setEnvVar RESTIC_REPOSITORY F:\Restic
    setEnvVar RESTIC_PASSWORD LPKcYEiF9CKRDxWf3B7o44SdAKZJfu34p8DeY7JWzRLjCmS9ji5mfjev5Jj2pJUt
```

<details>

<summary>Sample runs</summary>

<br />

```
$ autoshell run main
--------------------------------------------------------------------------------
Started at 2025-05-30T20:36:20.6646661+05:30
--------------------------------------------------------------------------------
Command ID: create-sql-dump
--------------------------------------------------------------------------------
[command output]
--------------------------------------------------------------------------------
Command ID: b2-backup-db
--------------------------------------------------------------------------------
[command output]
--------------------------------------------------------------------------------
Command ID: b2-backup-code
--------------------------------------------------------------------------------
[command output]
--------------------------------------------------------------------------------
Command ID: b2-backup-documents
--------------------------------------------------------------------------------
[command output]
--------------------------------------------------------------------------------
Command ID: ext-hdd-backup-db
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
Command ID: delete-sql-dump
--------------------------------------------------------------------------------
[command output]
--------------------------------------------------------------------------------
Ended at 2025-05-30T20:37:47.6888018+05:30 after 87024ms
--------------------------------------------------------------------------------
```

```
$ autoshell run restic ext-hdd snapshots -- --compact
--------------------------------------------------------------------------------
Started at 2025-05-30T20:43:15.7856331+05:30
--------------------------------------------------------------------------------
Command ID: ext-hdd
--------------------------------------------------------------------------------
repository 219fdcf6 opened (version 2, compression level auto)
ID        Time                 Host    Tags
---------------------------------------------
9476f199  2025-05-30 20:37:16  server
156ff78b  2025-05-30 20:37:28  server
41432ec6  2025-05-30 20:37:45  server
---------------------------------------------
3 snapshots
--------------------------------------------------------------------------------
Ended at 2025-05-30T20:43:20.5029457+05:30 after 4717ms
--------------------------------------------------------------------------------
```

</details>
