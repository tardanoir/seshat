# Seshat

A terminal UI SQL client for PostgreSQL, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Support

If you find Seshat useful, consider buying me a coffee:

<a href="https://buymeacoffee.com/tardanoir">
  <img src="assets/buymeacoffee.png" alt="Buy Me A Coffee" width="200">
</a>

## Features

- **Read-only SQL preview** with syntax highlighting and multi-statement support
- **External editor integration** via `$EDITOR` (Ctrl+E)
- **Statement selection** - cycle through statements with j/k, run the selected one with Ctrl+R
- **Saved queries** - persist and recall frequently used SQL
- **Parameterized templates** - SQL files with `{{variable}}` placeholders and TOML frontmatter
- **Schema browser** - collapsible sidebar with tables and columns
- **Multiple connections** - switch between configured PostgreSQL instances
- **File support** - Open your csv and json files and query them like SQL tables
- **Persistent sessions** - Your editor buffer, connection and layout are remembered per directory
- **Clipboard copy** - Copy a cell, a row, or a whole column straight out of the results table
- **Mouse support** - Click to focus a panel, click a table to drop a `SELECT` into the editor, scroll with the wheel
- **Self-update** - `seshat update` pulls the latest release and verifies its checksum before installing

## Elevator Pitch

Let's be honest: the state of SQL clients is embarrassing. On one end you've got psql, which renders your results like a ransom note the second a column is wider than 20 characters. You want to reuse a query? Hope you like scrolling through your shell history. You want to switch databases? Quit, retype the connection string from memory, fat-finger the port, try again.

On the other end you've got the GUI clients. DBeaver takes longer to load than your database takes to boot. pgAdmin spins up a whole web server so you can right-click through seventeen context menus. DataGrip wants $200 a year and half your RAM just to run a SELECT statement. You wanted a SQL client, not a second operating system. Each of these tools are great at what they're trying to accomplish, but there's too much bloat.

There's nothing in between. Either you suffer in a raw terminal or you surrender your entire machine to an application that treats running a query like launching the space shuttle.

Seshat is the thing in between. A fast, keyboard-driven TUI that connects to your Postgres databases, runs your queries, shows you clean results, and gets out of your way. Save queries, browse your schema, switch connections — all without leaving your terminal, all without waiting for anything to load. It does what you need and nothing you don't.

## Querying files

### What works

- All `SELECT` features: `SELECT`, `WHERE`, `ORDER BY`, `GROUP BY`, `HAVING`, `LIMIT`, `OFFSET`, `DISTINCT`
- Aggregations: `COUNT()`, `SUM()`, `AVG()`, `MIN()`, `MAX()`, `GROUP_CONCAT()`
- Joins: You won't have multiple tables in a single CSV, but you could self-join
- Subqueries: `SELECT * WHERE price > (SELECT AVG(price))`
- String functions: `LIKE`, `GLOB`, `LENGTH()`, `UPPER()`, `LOWER()`, `SUBSTR()`, `REPLACE()`, `TRIM()`
- Math: `ABS()`, `ROUND()`, arithmetic operators
- CASE expressions: `CASE WHEN ... THEN ... END`
- CTEs: `WITH cte AS (...) SELECT ...`
- Window functions: `ROW_NUMBER()`, `RANK()`, `LAG()`, `LEAD()`, etc.
- `CREATE INDEX`, `CREATE VIEW`, `INSERT`, `UPDATE`, `DELETE` (all in-memory, won't modify the file)
- Implicit FROM: `SELECT * WHERE age > 30` works — the FROM clause is auto-injected since there's only one table

### Limitations

- All columns are TEXT — there's no type inference. So WHERE age > 30 does a string comparison, not numeric. You need to cast: WHERE CAST(age AS INTEGER) > 30. I'll be working on a improvement for this very soon.
- In-memory only — any INSERT, UPDATE, DELETE, or schema changes are lost when you disconnect. The original file is never modified. You can write the results to a new file with `ctrl+x`
- Single table — one CSV = one table, named after the filename (e.g., sales_data.csv → table sales_data)
- Full file load — the entire file is read into memory on connect, so very large files (hundreds of MB+) may be slow to open or consume significant RAM
- No PostgreSQL-specific syntax — things like ILIKE, ::int casting, array operators, or JSONB functions won't work. It's SQLite, not Postgres.
- Flat structure only for JSON — JSON files must be an array of flat objects ([{"key": "val"}, ...]). Nested objects get stringified, not expanded into columns. Will also be working on a better solution. It's kinda hard through due to the different json structures.

The TEXT-column limitation is the biggest practical one. A common pattern would be:

```sql
SELECT name, CAST(price AS REAL) as price
WHERE CAST(price AS REAL) > 100
ORDER BY CAST(price AS REAL) DESC
```

## Support for new databases

I'll be adding support for Mysql and Mariadb soon. Other databases will be considered based on demand.

## Install

### Quick install (macOS / Linux)

```sh
curl -fsSL https://raw.githubusercontent.com/tardanoir/seshat/main/install.sh | sh
```

Detects your OS/arch and installs the latest release to `~/.local/bin`
(or `/usr/local/bin` as root). Override with `SESHAT_VERSION` / `SESHAT_BIN_DIR`.

### Homebrew (macOS / Linux)

```sh
brew tap tardanoir/tap
brew install seshat
```

### Arch (AUR)

```sh
yay -S seshat-bin
```

### Windows (Scoop)

```sh
scoop bucket add tardanoir https://github.com/tardanoir/scoop-bucket
scoop install seshat
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
persist_sessions = true  # remember state per directory; defaults to true

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
| y | Copy the cell under the cursor (results panel) |
| Y | Copy the whole row, tab-separated (results panel) |
| c | Copy the whole column, one value per line (results panel) |
| Ctrl+D | Delete selected query |
| Ctrl+C | Quit |

Mouse (no configuration needed):

| Action | Result |
|---|---|
| Click a panel | Focus it |
| Click a table in the sidebar | Append `select * from <table>;` to the editor and focus it |
| Click a section title | Switch sidebar section |
| Click a results cell | Move the cell cursor there |
| Scroll wheel | Scroll the results or sidebar under the pointer |

Clicking a table never overwrites the editor — the statement is appended below whatever is already there. Expanding a table's columns stays on `Enter`.

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

## Updating

```sh
seshat update          # download and install the latest release
seshat update --check  # just report whether one is available
```

The updater resolves the release asset for your OS and architecture, verifies its
SHA-256 against the published `checksums.txt`, and only then swaps the binary in
place (the old one is moved aside first and restored if the swap fails).

If seshat lives somewhere you can't write — a system prefix, or a package-manager
install — it stops before downloading and tells you to upgrade the way you
installed it (`brew upgrade`, `pacman -Syu`, `apt upgrade`, `scoop update seshat`).

Separately, seshat checks GitHub for a newer release at most once every 24h and
shows a `vX.Y.Z available` pill in the status bar.

## Sessions

Seshat remembers what you were doing in each directory. When you quit — and after
every query you run — the editor buffer, the active connection and the sidebar
state are written to a session file keyed on the current working directory. Start
seshat in that directory again and you pick up where you left off.

Sessions are keyed on the resolved working directory, so `~/work/api` and
`~/work/etl` keep separate buffers, and reopening either one restores only its own.

A connection passed on the command line still wins: `seshat data.csv` opens the
file, but the buffer from the last session in that directory is still restored.

Turn it off with `persist_sessions = false` in `config.toml`. Session files live in
`~/.config/seshat/sessions/` and can be deleted at any time — a missing one just
means seshat starts fresh in that directory.

## Directory Layout

```
~/.config/seshat/
  config.toml
  queries/
    my_query.sql
  templates/
    get_user.sql
  sessions/
    api-3f9a1c7b2d4e.json
```



