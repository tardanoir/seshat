# Seshat

A terminal UI SQL client for PostgreSQL, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Read-only SQL preview** with syntax highlighting and multi-statement support
- **External editor integration** via `$EDITOR` (Ctrl+E)
- **Statement selection** - cycle through statements with j/k, run the selected one with Ctrl+R
- **Saved queries** - persist and recall frequently used SQL
- **Parameterized templates** - SQL files with `{{variable}}` placeholders and TOML frontmatter
- **Schema browser** - collapsible sidebar with tables and columns
- **Multiple connections** - switch between configured PostgreSQL instances

## Install

```sh
go build -o seshat .
```

## Configuration

Config lives at `~/.config/seshat/config.toml`. A default one is created on first run.

```toml
default_connection = "local"
editor = "nvim"  # falls back to $EDITOR, then "vi"

[connections.local]
host = "localhost"
port = 5432
database = "postgres"
user = "postgres"
password = ""
sslmode = "disable"

[connections.production]
host = "prod.example.com"
port = 5432
database = "app_db"
user = "app_user"
password = "$DB_PASSWORD"  # env var expansion supported
sslmode = "require"
```

## Saved Queries

SQL files stored in `~/.config/seshat/queries/`. Save the current buffer with Ctrl+W, browse and load from the sidebar (press `1`), delete with Ctrl+D.

## Templates

SQL files with TOML frontmatter stored in `~/.config/seshat/templates/`. Open the template picker with Ctrl+T, fill in variable values, and the result is loaded into the preview pane.

Template format:

```sql
+++
name = "Get User By ID"
description = "Fetches a user record"

[variables.user_id]
description = "The user ID to lookup"
default = "1"

[variables.table_name]
description = "Table name"
default = "users"
+++

SELECT * FROM {{table_name}} WHERE id = {{user_id}};
```

Variables use `{{name}}` syntax and are substituted before execution.

## Keybindings

| Key | Action |
|---|---|
| Ctrl+E | Open `$EDITOR` to edit SQL |
| Ctrl+R | Execute selected statement |
| Ctrl+W | Save current SQL as named query |
| Ctrl+T | Open template picker |
| Ctrl+N | Switch connection |
| Ctrl+\ | Toggle sidebar |
| Tab | Cycle focus (sidebar / preview / results) |
| j/k | Navigate within focused panel |
| 1/2/3 | Switch sidebar section (queries/templates/tables) |
| Ctrl+D | Delete selected query |
| Ctrl+C | Quit |

All keybindings are configurable via `config.toml`:

```toml
[keybindings]
execute = "ctrl+r"
editor = "ctrl+e"
save = "ctrl+w"
template = "ctrl+t"
connection = "ctrl+n"
toggle_sidebar = "ctrl+\\"
tab = "tab"
shift_tab = "shift+tab"
quit = "ctrl+c"
delete = "ctrl+d"
```

Only include the keys you want to override — unset keys use the defaults above.

## Directory Layout

```
~/.config/seshat/
  config.toml
  queries/
    my_query.sql
  templates/
    get_user.sql
```
