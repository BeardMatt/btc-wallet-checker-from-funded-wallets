# Craigpwn — BTC Wallet Checker From Funded Wallets

A high-performance, distributed Bitcoin private key search tool (also known internally as **btcfind**).

It randomly generates private keys, derives Bitcoin addresses across multiple formats, and checks them against a large index of "funded" addresses (real on-chain addresses that have received BTC).

The project focuses on extreme optimization of the search hot loop, caching, indexing, and distributed execution across machines.

> **Note:** The probability of success with random search is astronomically low. This tool is primarily for research, performance engineering, and distributed systems experimentation.

## Features

- **Multi-format address derivation**: Legacy (1...), Legacy compressed/uncompressed, P2SH (3...), Native SegWit (bc1q...), Taproot (bc1p...)
- **High-performance hot loop**: Optimized key generation, hashing, and address encoding (using btcsuite + custom SIMD-accelerated crypto where possible)
- **Fast funded lookup**: Bloom filter + memory-mapped index for billions of address hashes
- **Distributed cluster mode**: Coordinator + worker architecture for scaling across many machines
- **Resumable searches**: `--forever` mode with automatic session checkpointing (`btcfind.session`)
- **Flexible filtering**: Minimum balance threshold, specific address formats, simulation/injection modes for testing
- **Rich output**: Configurable formats (human, JSON, TSV, etc.), live progress, quiet/verbose modes
- **Benchmark-driven development**: Extensive planning system with recorded performance history

## Installation / Build

### Prerequisites
- Go 1.25+ (see `go.mod`)
- (Optional) For full data: significant RAM and disk for funded index caches

### Build from source

```bash
git clone https://github.com/BeardMatt/btc-wallet-checker-from-funded-wallets.git
cd btc-wallet-checker-from-funded-wallets

# Main searcher
go build -o btcfind .

# Coordinator (for distributed mode)
go build -o btcfind-coordinator ./cmd/coordinator

# Worker (for distributed mode)
go build -o btcfind-worker ./cmd/worker
```

Pre-built binaries are also available in the project root after checkout (`btcfind`, `btcfind-coordinator`, `btcfind-worker`).

## Usage

### Basic Search

```bash
# Try 1 million random keys with 8 threads
./btcfind 1000000 8

# With minimum balance filter (in satoshis) and specific formats
./btcfind 5000000 16 --min-balance 100000 --formats segwit,taproot
```

**Usage:**

```
./btcfind <number of wallets to try> [threads] [flags]
./btcfind --forever [threads] [flags]
```

### Key Flags

- `--forever` — Run continuously until interrupted; checkpoints progress
- `--min-balance SATS` — Only report wallets with at least this balance (default: 30000)
- `--formats LIST` — Comma-separated list of address types (legacy, segwit, p2sh, taproot, all, ...)
- `--checkpoint-interval DURATION` — How often to save session (default: 60s)
- `--reset-session` — Start fresh (ignore existing `btcfind.session`)
- `--no-color` — Disable ANSI colors and live progress bar
- `--quiet` / `--verbose` — Control output verbosity
- Simulation flags (`--simulate-hit`, `--inject-index-hit`) — For testing the full hit/output pipeline

### Distributed / Cluster Mode

1. Start the coordinator (requires TLS certs or `--insecure` for testing):
   ```bash
   ./btcfind-coordinator --listen :8080 --cert certs/server.pem --key certs/server.key
   ```

2. Start one or more workers:
   ```bash
   ./btcfind-worker --coordinator https://coordinator-ip:8080 --threads 8
   ```

Workers pull work from the coordinator and report hits. This enables massive horizontal scaling.

See the `cluster/` and `worker/` directories for implementation details.

## Data Files

The tool requires a list of funded addresses.

- `funded.tsv` — The master list (can be auto-downloaded)
- `funded.cache`, `funded.*.cache` — Optimized binary/mmap caches (generated automatically)

Large caches (tens of GB when fully built) live in the project root. They are gitignored.

The project can automatically download and build the index on first use.

## Performance

This project is heavily optimized. All significant changes are tracked in `plans/benchmarks.json` and must be accompanied by a standardized benchmark:

```bash
./scripts/run-benchmark.sh
# or
go build -o btcfind .
time ./btcfind 50000 8
```

Key metrics tracked: keys/sec, hot loop time, startup time, wall time, and relative improvement.

See `plans/manifest.json` and `plans/benchmarks.json` for the current optimization roadmap and history.

## Project Structure

- `cli.go` — Main CLI and argument parsing
- `search/` — Core search logic and workers
- `funded*/` — Funded address loading, indexing, caching, and downloading
- `bitcoin/` — Address derivation and format handling (forked/custom btcd bits)
- `cluster/`, `worker/`, `coordinator/` — Distributed execution components
- `plans/` — Detailed optimization plans and benchmark history (used by AI agents)
- `cryptogen.go`, `simulate.go`, etc. — Key generation and testing utilities

## Warnings & Disclaimer

- **Extremely low odds**: Randomly generating keys that match funded addresses is effectively impossible at scale. This is a research/performance project.
- **Resource intensive**: Full index builds and long searches can consume large amounts of CPU, RAM, and disk.
- **Legal/Ethical**: Only use on your own keys or for legitimate research. Attempting to access funds you do not control may be illegal in your jurisdiction.
- The presence of a "hit" in simulation or injected modes does **not** mean real funds have been found.

## Development

This project follows a strict plan-driven workflow (see `AGENTS.md` and `plans/`):

1. Read the current plan from `plans/manifest.json`
2. Implement only what the plan describes
3. Benchmark after changes
4. Record results and update manifest

Contributions that follow this process (especially performance improvements) are welcome.

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

## Disclaimer

**Critical legal and usage warnings apply to this software.**

This is a research and educational tool for exploring Bitcoin cryptography and high-performance search techniques. The probability of randomly discovering private keys for funded wallets is extremely low.

**You must read and understand the full [DISCLAIMER](DISCLAIMER.md) before using this software.** By downloading, building, or running this project, you acknowledge and accept all terms and restrictions contained in the disclaimer.

---

*Part of the Craigpwn / btcfind family of Bitcoin research tools.*
