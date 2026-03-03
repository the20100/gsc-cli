# gsc — Google Search Console CLI

A fast, agent-friendly CLI for the [Google Search Console API v1](https://developers.google.com/webmaster-tools/v1/api_reference_index).

Outputs **JSON when piped** (for scripting/agents) and **human-readable tables** in a terminal.

---

## Install

```bash
git clone https://github.com/the20100/gsc-cli
cd gsc-cli
go build -o gsc .
mv gsc /usr/local/bin/
```

## Update

```bash
gsc update
```

---

## Authentication

The CLI uses **OAuth2** (Desktop app flow). You need a Google Cloud project with the Search Console API enabled.

**One-time setup:**

1. Go to [Google Cloud Console → APIs & Credentials](https://console.cloud.google.com/apis/credentials)
2. Create credentials → **OAuth client ID** → **Desktop app**
3. Add `http://localhost:8080` as an authorized redirect URI
4. Download the `credentials.json` file

```bash
# Option A: provide the downloaded file
gsc auth setup --credentials /path/to/credentials.json

# Option B: pass flags
gsc auth setup --client-id <id> --client-secret <secret>

# Option C: env vars (client credentials only, still needs oauth flow for token)
export GOOGLE_CLIENT_ID=...
export GOOGLE_CLIENT_SECRET=...
gsc auth setup

# Option D: remote/VPS — no browser available
gsc auth setup --credentials /path/to/credentials.json --no-browser
```

With `--no-browser`: the CLI prints the OAuth URL. Open it in a local browser, authorize, then copy the full redirect URL from the address bar and paste it into the terminal (the page will fail to load — that's expected).

Config stored at:
- macOS: `~/Library/Application Support/g-search-console/config.json`
- Linux: `~/.config/g-search-console/config.json`

---

## Commands

### `gsc info`
Show binary location, config path, and auth status.

### `gsc auth setup / status / logout`
Manage OAuth2 authentication.

---

### `gsc sites list`
List all verified site properties.
```bash
gsc sites list
gsc sites list --json
```

### `gsc sites get <url>`
Get details for a site property.
```bash
gsc sites get https://example.com
```

### `gsc sites add <url>` / `gsc sites delete <url>`
Add or remove a site property.

---

### `gsc analytics query`
Query search performance data (clicks, impressions, CTR, position).

```bash
# Required flags: --site, --start, --end
gsc analytics query --site https://example.com --start 2024-01-01 --end 2024-01-31

# With dimensions
gsc analytics query --site https://example.com \
  --start 2024-01-01 --end 2024-01-31 \
  --dimensions query,page,country \
  --limit 100

# Image search
gsc analytics query --site https://example.com \
  --start 2024-01-01 --end 2024-01-31 \
  --search-type image --dimensions date
```

**Flags:**

| Flag | Default | Description |
|---|---|---|
| `--site` | *(required)* | Site URL |
| `--start` | *(required)* | Start date (YYYY-MM-DD) |
| `--end` | *(required)* | End date (YYYY-MM-DD) |
| `--dimensions` | — | Comma-separated: `date,query,page,country,device,searchAppearance,hour` |
| `--search-type` | `web` | `web`, `image`, `video`, `news`, `discover`, `googleNews` |
| `--data-state` | — | `final`, `all`, `hourlyAll` |
| `--limit` | `1000` | Max rows (1–25000) |
| `--start-row` | `0` | Pagination offset |

---

### `gsc sitemaps list <site-url>`
List all sitemaps for a site.
```bash
gsc sitemaps list https://example.com
```

### `gsc sitemaps get <site-url> <feedpath>`
Get sitemap details.
```bash
gsc sitemaps get https://example.com https://example.com/sitemap.xml
```

### `gsc sitemaps submit <site-url> <feedpath>`
Submit a sitemap.
```bash
gsc sitemaps submit https://example.com https://example.com/sitemap.xml
```

### `gsc sitemaps delete <site-url> <feedpath>`
Delete a sitemap.

---

### `gsc inspect <url> --site <site>`
Inspect URL indexing status, coverage, mobile usability, and rich results.
```bash
gsc inspect https://example.com/page --site https://example.com
gsc inspect https://example.com/page --site https://example.com --language fr
gsc inspect https://example.com/page --site https://example.com --json
```

---

### `gsc mobile-test <url>`
Run a mobile-friendly test for a URL.
```bash
gsc mobile-test https://example.com
gsc mobile-test https://example.com --json
```

---

## Global Flags

| Flag | Description |
|---|---|
| `--json` | Force JSON output |
| `--pretty` | Force pretty-printed JSON (implies `--json`) |

---

## Tips

- **Pagination**: use `--start-row` + `--limit` to page through large analytics result sets.
- **URL format**: site URLs must match Search Console exactly (including trailing slash if registered that way).
- **Piped output**: JSON is emitted automatically when stdout is not a TTY — useful for `jq` pipelines.
  ```bash
  gsc analytics query --site https://example.com --start 2024-01-01 --end 2024-01-31 --dimensions query \
    | jq '.rows | sort_by(-.clicks) | .[0:10]'
  ```
