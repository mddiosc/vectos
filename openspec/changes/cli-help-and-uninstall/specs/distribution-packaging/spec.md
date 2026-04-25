## ADDED Requirements

### Requirement: The Vectos installer SHALL support uninstall
The installer script SHALL support a `--uninstall` flag that removes the installed Vectos binary from the installation directory.

#### Scenario: Uninstall removes the binary
- **WHEN** the user runs the installer with `--uninstall`
- **THEN** the script SHALL remove the `vectos` binary from `DEST_DIR` and confirm removal

#### Scenario: Uninstall when binary does not exist
- **WHEN** the user runs `--uninstall` and no binary is found at `DEST_DIR/vectos`
- **THEN** the script SHALL exit with a clear message indicating nothing was found to remove

#### Scenario: Uninstall shows manual purge guidance
- **WHEN** uninstall completes successfully
- **THEN** the script SHALL print the list of data and config directories the user may want to remove manually, without removing them automatically
