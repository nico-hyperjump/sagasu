# Sagasu CLI Reference

Command-line interface for Sagasu.

## Commands

### server

Start the HTTP API server.

```bash
sagasu server [--config PATH]
```

| Flag     | Default                           | Description       |
| -------- | --------------------------------- | ----------------- |
| --config | /usr/local/etc/sagasu/config.yaml | Config file path. |

**Example:**

```bash
sagasu server
sagasu server --config ./config.yaml
```

---

### search

Run a hybrid search from the command line.

```bash
sagasu search [flags] <query>
```

| Flag              | Default      | Description            |
| ----------------- | ------------ | ---------------------- |
| --config          | (see server) | Config file path.      |
| --limit           | 10           | Number of results.     |
| --keyword-weight  | 0.5          | Keyword score weight.  |
| --semantic-weight | 0.5          | Semantic score weight. |

**Examples:**

```bash
sagasu search "machine learning algorithms"
sagasu search --limit 20 "neural networks"
sagasu search --keyword-weight 0.7 --semantic-weight 0.3 "search engine"
```

---

### index

Index a document from a file.

```bash
sagasu index [flags] <file>
```

| Flag     | Default      | Description       |
| -------- | ------------ | ----------------- |
| --config | (see server) | Config file path. |
| --title  | ""           | Document title.   |

**Examples:**

```bash
sagasu index document.txt
sagasu index --title "My Notes" notes.txt
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
