# Autoshell

Autoshell is a command-line utility facilitating automatic execution of shell commands.

[![Latest Release](https://img.shields.io/github/v/release/shibijm/autoshell?label=Latest%20Release)](https://github.com/shibijm/autoshell/releases/latest)
[![Build Status](https://img.shields.io/github/actions/workflow/status/shibijm/autoshell/release.yml?label=Build&logo=github)](https://github.com/shibijm/autoshell/actions/workflows/release.yml)

## Download

Downloadable builds are available on the [releases page](https://github.com/shibijm/autoshell/releases).

## Usage

```text
Usage:
  autoshell [command]

Available Commands:
  decrypt     Decrypt the config file
  encrypt     Encrypt the config file
  run         Run a workflow

Flags:
  -c, --config string   config file path (default "config.yml")
  -h, --help            help for autoshell
  -v, --version         version for autoshell

Use "autoshell [command] --help" for more information about a command.
```

## Configuration

### Encryption

Config file encryption employs AES-256 in GCM mode. The encryption key is derived using Argon2id.

Instances of `$DP` within passwords will be substituted with a device pass, derived using `sha256(machineId + devicePassSeed + salt)`.

The `run` command will attempt to automatically decrypt the config file using the device pass before prompting for manual password input. Such an attempt would be successful only on the machine that initially encrypted the config file. Config files used for fully automated runs can be obfuscated by using this feature.

If a config file is marked as protected, the `decrypt` command will refuse to save the decrypted data to disk if the decryption password contains `$DP`. In such cases, `$DP` has to be substituted with its actual value, which is displayed only once during encryption.

<details>

<summary>Example</summary>

<br />

```text
$ cat config.yml
protected: true
workflows:
  hello: runCommand - echo Hello world

$ autoshell encrypt
Password: $DP (hidden input)
Confirm Password: $DP (hidden input)
$DP = e2a7e8ee355591e1d5f80d19bcb34e91936e08b061b699c7682aadd83c6e9e0c
Config file is marked as protected and hence cannot be saved decrypted without substituting "$DP"
Config file encrypted successfully

$ cat config.yml
}%␦□□)□+w□□*□□`□□Z□□□□B\□□a□□`P_sK□□V□WgYFw]□dD□DyD□B□P;b□\□@-□B□□□y□'□□□

$ autoshell run hello
--------------------------------------------------------------------------------
Started at 2025-05-11T19:18:13.1756292+05:30
--------------------------------------------------------------------------------
Hello world
--------------------------------------------------------------------------------
Ended at 2025-05-11T19:18:13.2735076+05:30 after 97ms
--------------------------------------------------------------------------------

$ autoshell decrypt
Password: $DP (hidden input)
Error: file is protected and the password contains "$DP"

$ autoshell decrypt
Password: e2a7e8ee355591e1d5f80d19bcb34e91936e08b061b699c7682aadd83c6e9e0c (hidden input)
Config file decrypted successfully
```

</details>

### Actions

- `runWorkflow <workflow> [args...]`
- `setEnvVar <name> <value>`
- `setGlobalVar <name> <value>`
- `setLocalVar <name> <value>`
- `runCommand <commandId> <command> [args...]`
- `setLogFile <path>`
- `addReporter <kind> <endpoint>`
- `setIgnoredExitCodes <codes: []int>`
- `print [args...]`
- `shiftArgs`

Append modifiers to action names using `!`. Separate multiple modifiers with commas.

- `*!W`: Restrict execution to Windows
- `*!L`: Restrict execution to Linux
- `runCommand!hideCommandId`
- `runCommand!retries=n`: Retry`n` times on failure
- `runCommand!ignoreFailures`: Ignore failures (after retries)

### Variable Substitution

`$x` gets substituted with the value of variable `x`.

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
    runWorkflow setup-restic $1
    shiftArgs
    runCommand restic-$resticDestination $restic $@
  setup-restic: |-
    setEnvVar RCLONE_CONFIG notfound
    setEnvVar RCLONE_FAST_LIST true
    setEnvVar RCLONE_BWLIMIT 8M
    setIgnoredExitCodes [3]
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

```text
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

```text
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
