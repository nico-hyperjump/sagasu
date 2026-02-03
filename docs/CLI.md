# Sagasu CLI Reference

Command-line interface for Sagasu.

## Commands

### server

Start the HTTP API server.

```bash
sagasu server [--config PATH] [--debug]
```

| Flag     | Default                           | Description                                                    |
| -------- | --------------------------------- | -------------------------------------------------------------- |
| --config | /usr/local/etc/sagasu/config.yaml | Config file path.                                              |
| --debug  | false                             | Enable debug logging (directory changes, file indexing, etc.). |

**Example:**

```bash
sagasu server
sagasu server --config ./config.yaml
sagasu server --debug
```

---

### search

Run a hybrid search from the command line.

```bash
sagasu search [flags] <query>
```

| Flag              | Default               | Description                                                                                       |
| ----------------- | --------------------- | ------------------------------------------------------------------------------------------------- |
| --config          | (see server)          | Config file path.                                                                                 |
| --server          | http://localhost:8080 | Server URL. Use `--server ""` to use direct storage (config DB/Bleve) when server is not running. |
| --limit           | 10                    | Number of results.                                                                                |
| --min-score       | 0.05                  | Minimum score threshold.                                                                          |
| --keyword-weight  | 0.5                   | Keyword score weight.                                                                             |
| --semantic-weight | 0.5                   | Semantic score weight.                                                                            |

**Examples:**

```bash
sagasu search "machine learning algorithms"
sagasu search --limit 20 "neural networks"
sagasu search --min-score 0.1 "raosan"
sagasu search --keyword-weight 0.7 --semantic-weight 0.3 "search engine"
```

---

### index

Index a file or a directory. For a directory, only files whose extension is in `watch.extensions` (from config) are indexed. Supports plain text and binary formats:

- **Plain text**: `.txt`, `.md`, `.rst` (content used as-is)
- **PDF**: `.pdf` (text extracted from pages)
- **Word**: `.docx`, `.odt`, `.rtf` (text extracted)
- **Excel**: `.xlsx` (cell values extracted from all sheets)

```bash
sagasu index [flags] <file-or-directory>
```

| Flag     | Default      | Description                                                       |
| -------- | ------------ | ----------------------------------------------------------------- |
| --config | (see server) | Config file path.                                                 |
| --title  | ""           | Document title (unused; document title is derived from filename). |

**Examples:**

```bash
sagasu index document.txt
sagasu index report.pdf
sagasu index spreadsheet.xlsx
sagasu index ./dev/sample
```

---

### delete

Delete a document by ID.

```bash
sagasu delete [flags] <document-id>
```

| Flag     | Default      | Description       |
| -------- | ------------ | ----------------- |
| --config | (see server) | Config file path. |

**Example:**

```bash
sagasu delete doc-123
```

---

### watch

Manage watched directories (requires server running).

```bash
sagasu watch add <path>
sagasu watch remove <path>
sagasu watch list
```

| Subcommand | Description                             |
| ---------- | --------------------------------------- |
| add        | Add directory to watch and index files. |
| remove     | Stop watching directory.                |
| list       | List watched directories.               |

| Flag     | Default               | Description |
| -------- | --------------------- | ----------- |
| --server | http://localhost:8080 | Server URL. |

**Examples:**

```bash
sagasu watch add /path/to/docs
sagasu watch add --server http://localhost:9000 ~/notes
sagasu watch list
sagasu watch remove /path/to/docs
```

---

### version

Print version.

```bash
sagasu version
# or
sagasu --version
sagasu -v
```

---

### help

Print usage.

```bash
sagasu help
sagasu --help
sagasu -h
```

---

## Config file

Default path: `/usr/local/etc/sagasu/config.yaml`

Override with `--config` on any command. See the repository `config.yaml.example` for all options.
