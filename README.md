[![Buy me a pizza](https://img.buymeacoffee.com/button-api/?text=Buy me a pizza&emoji=🍕&slug=tardanoir&button_colour=5F7FFF&font_colour=ffffff&font_family=Inter&outline_colour=000000&coffee_colour=FFDD00)](https://www.buymeacoffee.com/tardanoir)

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

## Elevator Pitch

Let's be honest: the state of SQL clients is embarrassing. On one end you've got psql, which renders your results like a ransom note the second a column is wider than 20 characters. You want to reuse a query? Hope you like scrolling through your shell history. You want to switch databases? Quit, retype the connection string from memory, fat-finger the port, try again.

On the other end you've got the GUI clients. DBeaver takes longer to load than your database takes to boot. pgAdmin spins up a whole web server so you can right-click through seventeen context menus. DataGrip wants $200 a year and half your RAM just to run a SELECT statement. You wanted a SQL client, not a second operating system. Each of these tools are great at what they're trying to accomplish, but there's too much bloat.

There's nothing in between. Either you suffer in a raw terminal or you surrender your entire machine to an application that treats running a query like launching the space shuttle.

Seshat is the thing in between. A fast, keyboard-driven TUI that connects to your Postgres databases, runs your queries, shows you clean results, and gets out of your way. Save queries, browse your schema, switch connections — all without leaving your terminal, all without waiting for anything to load. It does what you need and nothing you don't.

## Support for new databases

I'm currently supporting PostgreSQL and SQLite, and my plan is to keep it that way for now. I'd love the idea to have all databases supported, but the truth is that I'm currently not a user of any db's other then these two. I might add MySQL (and MariaDB for that matter), but that's not a promisse. The goal of this project is to do a tool that I really want to use, and also to learn go in the process. I'll keep doing the features that interest me.

## Install

### Homebrew (macOS / Linux)

```sh
brew tap tardanoir/tap
brew install seshat
```

### Debian / Ubuntu (.deb)

Download the `.deb` from the [latest release](https://github.com/tardanoir/seshat/releases/latest), then:

```sh
sudo dpkg -i seshat_*.deb
```

### Fedora / RHEL (.rpm)

Download the `.rpm` from the [latest release](https://github.com/tardanoir/seshat/releases/latest), then:

```sh
sudo rpm -i seshat_*.rpm
```

### Go

```sh
go install github.com/tardanoir/seshat@latest
```

### From source

```sh
git clone https://github.com/tardanoir/seshat.git
cd seshat
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
ssl_mode = "disable"

[connections.production]
type = "postgres"  # optional, defaults to "postgres"
host = "prod.example.com"
port = 5432
database = "app_db"
user = "app_user"
password = "$DB_PASSWORD"  # env var expansion supported
ssl_mode = "require"

[connections.mydb]
type = "sqlite"
path = "/path/to/database.db"
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
