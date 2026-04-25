## 1. English-only CLI

- [x] 1.1 Translate all user-visible strings in `cmd/vectos/main.go` to English
- [x] 1.2 Translate code comments in `cmd/vectos/main.go` to English

## 2. CLI Help System

- [x] 2.1 Implement a central `printHelp()` function with global usage for all subcommands
- [x] 2.2 Implement per-subcommand help for `index`, `search`, `status`, `mcp`, `setup`, `version`
- [x] 2.3 Handle `--help` / `-h` before argument parsing so it always works
- [x] 2.4 Add `help` and `help <subcommand>` subcommands

## 3. Installer Uninstall

- [x] 3.1 Add `--uninstall` to `scripts/install.sh` that removes the binary from `DEST_DIR`
- [x] 3.2 Show manual purge guidance after successful uninstall

## 4. Documentation

- [x] 4.1 Update README with uninstall instructions

## 5. Validation

- [x] 5.1 Validate `vectos --help`, `vectos -h`, `vectos help`
- [x] 5.2 Validate `vectos help <subcommand>` and `vectos <subcommand> --help`
- [x] 5.3 Validate `scripts/install.sh --uninstall` when binary exists
- [x] 5.4 Validate `scripts/install.sh --uninstall` when binary does not exist
- [x] 5.5 Confirm build is clean
