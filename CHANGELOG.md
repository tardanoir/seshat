# Changelog

## [0.5.0] - 2026-04-23

### Added
- Autocomplete popup in the query editor with keyword, table, and column suggestions
- Context-aware ranking: tables first after `FROM`/`JOIN`, columns first after `WHERE`/`SELECT`/`ON`/`HAVING`/etc.
- `Tab` expands to the longest common prefix when multiple candidates share one, then accepts on the next press
- `Enter` accepts the selected suggestion; `Esc` dismisses; `↑`/`↓` and `Ctrl+P`/`Ctrl+N` navigate
- Eager column loading on connect so autocomplete has the full column pool without needing to expand tables in the sidebar
- SSH tunnel support for connecting to Postgres instances over a bastion host

### Changed
- Refreshed color theme across panels, pills, and status bar for better contrast
- Connection status indicator recolored

### Fixed
- Modals no longer render on a hard-black overlay that mismatched the terminal theme

## [0.4.1] - 2026-03-17

### Fixed
- Numeric columns now display with correct formatting
- Query execution performance improvements

## [0.4.0] - 2026-03-16

### Added
- File support: query local CSV and JSON files directly with SQL

## [0.3.1] - 2026-03-14

### Added
- Configurable row limit for query results
- In-app notification when a newer version of Seshat is available

## [0.3.0] - 2026-03-14

### Added
- SQLite connection support

## [0.2.2] - 2026-03-13

### Fixed
- Connection strings with special characters in passwords are now parsed correctly

## [0.2.1] - 2026-03-13

### Added
- Debian (`.deb`) package and Homebrew tap distribution

## [0.2.0] - 2026-03-13

### Added
- Copy cell (`y`) and row (`Y`) to clipboard from results view
- Export results to CSV or JSON (`Ctrl+X`)
- Query history persisted across sessions (sidebar section 4)
- Help modal with all keybindings (`?`)
- `--version` flag
- GoReleaser config for automated releases
- External GUI editor support (VSCode, Cursor, Zed, Sublime, etc.)
- Suspend keybinding (`Ctrl+Z`)

### Changed
- Status bar hints are now context-aware (results vs query view)
- Column cursor in results view (`h/l` moves columns, `Home/End` jumps to first/last)
- `vim_mode` is now optional (defaults to textarea mode)

### Fixed
- Sidebar height overflow when viewing history section
- CSV/JSON export now writes proper nulls instead of literal "NULL" strings

## [0.1.0] - 2026-03-12

### Added
- Initial release
- PostgreSQL connection with multiple named connections
- Query editor with vim and textarea modes
- Schema browser (tables, columns)
- Saved queries and parameterized templates
- Multi-statement execution
- Configurable keybindings via TOML
- External editor integration (terminal editors)
