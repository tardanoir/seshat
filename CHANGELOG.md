# Changelog

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
