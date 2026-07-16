# CQLizer

CQLizer is a data-driven CQL (Corpus Query Language) writing helper tool for linguistic corpus analysis. It uses machine learning models to predict query performance and help users write efficient CQL queries.

## Features

- CQL query parsing and AST generation using PEG grammar
- Machine learning-based query performance prediction (Random Forest, Neural Network, XGBoost)
- Data import from KonText log files
- Multiple interfaces: CLI, REPL, and API server

## Requirements

- Go 1.24+
- [pigeon](https://github.com/mna/pigeon) parser generator (for development)

## Installation

```bash
# Install dependencies
make tools

# Build the binary
make build
```

## Usage

### CLI Commands

```bash
# Show version information
cqlizer version

# Start interactive REPL
cqlizer repl <model_file.json>

# Extract features from query logs
cqlizer featurize config.json logfile.jsonl output.msgpack

# Train a model
cqlizer learn [options] config.json features_file.msgpack

# Start API server
cqlizer server config.json

# Start MCP server (experimental)
cqlizer mcp-server config.json
```

### Learning Options

```bash
# Random Forest
cqlizer learn -model rf -num-trees 100 config.json features.msgpack

# Neural Network
cqlizer learn -model nn config.json features.msgpack
```

#### XGBoost Model

For XGBoost, the `learn` action extracts features into a format compatible with LightGBM. After running the extraction, use the Python script to train the model.

First, set up a Python virtual environment with the required dependencies:

```bash
python3 -m venv venv
source venv/bin/activate
pip install lightgbm==3.3.5 msgpack numpy scikit-learn
```

Then run the training:

```bash
# Step 1: Extract features
cqlizer learn -model xg config.json features.msgpack

# Step 2: Train the model using Python
python scripts/learnxgb.py --input ./cql_features.v3.17.msgpack --output ./cql_model.v3.17.model.xg.txt
```

Use `cqlizer help <command>` for detailed information about specific commands.

## Configuration

Most actions need a proper JSON configuration file. You can use the sample configuration file `conf-sample.json` as a base.

Note: `conf-sample.json` is configured with testing XGBoost model files located in the `testdata/` directory.

### MCP configuration

The `mcp-server` action is configured entirely via environment variables (rather than
command-line flags):

- **`LOG_PATH`** - Path to the log file. If unset, defaults to
  `$XDG_STATE_HOME/cqlizer/cqlizer.log` (or `~/.local/state/cqlizer/cqlizer.log` if
  `XDG_STATE_HOME` is not set). Set to `stderr` to log to stderr instead of a file.
- **`LOG_LEVEL`** - Logging verbosity: `debug`, `info`, `warning`/`warn`, or `error`.
  Defaults to `info`.
- **`REGISTRY_PATH`** - Path to the Manatee corpora registry directory, used to look up
  corpus structure (positional attributes, structures, structural attributes) for the
  `cql_validate` and `get_corpus_structure` tools. Defaults to
  `/var/opt/manatee/registry`.
- **`MODE`** - Transport mode for the MCP server: `stdio` or `http`. Defaults to `stdio`.
- **`LISTEN_ADDRESS`** - Address the server listens on when `MODE=http` (e.g.
  `localhost:8080`). Ignored in `stdio` mode. Defaults to `localhost:8080`.
- **`CQLIZER_CONF_PATH`** - Path to the JSON configuration file (same format as used by
  the other `cqlizer` actions, see `conf-sample.json`). It provides the RF/XGBoost model
  ensemble (enables the `cql_eval_complexity` tool) and per-corpus properties such as
  size and language. This is effectively required - without it, model-based query
  evaluation is unavailable.
- **`CORPUS_STRUCTURE_TOOL_ENABLED`** - Set to `1` to expose the `get_corpus_structure`
  tool, which lets clients list all positional attributes and structures of a corpus.
  Disabled by default.

### Server-Specific Model Training

**Important**: The model is trained on data from a specific server which has its own load and performance characteristics. For proper deployment in production, it is necessary to train the model on data obtained from your own server to ensure accurate performance predictions.

The most affected feature is corpus size, as e.g. on a less powerful machine than the one we used to train the sample model, there will likely be too many false negatives (and vice versa - a more powerful server will cause more false positives).

## Development

```bash
# Generate parser from PEG grammar
make generate

# Run tests
go test ./...

# Build everything
make all
```
