# Hedgerules Architecture

## Overview

Hedgerules is a Go CLI tool that manages CloudFront Functions and Key Value Store (KVS) data for Hugo static sites deployed to AWS CloudFront + S3. It replaces the existing Python scripts (`kvs-request.py`, `kvs-response.py`) and adds CloudFront Function management.

The tool reads a built Hugo site, extracts redirect rules and custom header definitions, validates them against CloudFront KVS constraints, and syncs the data to AWS. It also deploys the CloudFront Functions that consume this KVS data at the edge.

---

## 1. Package / Directory Layout

```
hedgerules/
  cmd/
    hedgerules/
      main.go              # Entry point, wires up CLI
  internal/
    config/
      config.go            # Config file loading + defaults
    hugo/
      directories.go       # Scan Hugo output dirs for index redirects
      redirects.go         # Parse Hugo _redirects file
      headers.go           # Parse _cfheaders.json
    kvs/
      types.go             # KVSData, KVSEntry types
      validate.go          # KVS constraint validation
      sync.go              # Diff + sync logic (put/delete)
    functions/
      embed.go             # go:embed for JS function code
      deploy.go            # Create/update CloudFront Functions via API
    aws/
      clients.go           # AWS SDK client factory
  functions/
    viewer-request.js      # CloudFront Function: redirects + index rewrite
    viewer-response.js     # CloudFront Function: custom response headers
  go.mod
  go.sum
```

### Package responsibilities

| Package | Responsibility |
|---|---|
| `cmd/hedgerules` | CLI entry point, flag parsing, command dispatch |
| `internal/config` | Load TOML/YAML config, merge with CLI flags |
| `internal/hugo` | Parse Hugo build output: directories, `_redirects`, `_cfheaders.json` |
| `internal/kvs` | KVS data types, validation against constraints, diff-and-sync to AWS |
| `internal/functions` | Embed JS source, prepend `kvsId` constant, deploy to CloudFront Functions API |
| `internal/aws` | AWS SDK v2 client construction with shared config |
| `functions/` | Raw JS source files (embedded at compile time) |

---

## 2. CLI Command Structure

Use **stdlib `flag`** package only. No third-party CLI framework. The tool has a small surface area (one primary command) and doesn't justify a dependency like cobra.

```
hedgerules deploy [flags]
hedgerules validate [flags]
hedgerules version
```

### `hedgerules deploy`

The primary command. Does everything: parse, validate, sync KVS, deploy functions.

```
hedgerules deploy \
  --output-dir public/ \
  --redirects-kvs-name mysite-redirects \
  --headers-kvs-name mysite-headers \
  --request-function-name mysite-viewer-request \
  --response-function-name mysite-viewer-response \
  --dry-run
```

Flags:
| Flag | Required | Description |
|---|---|---|
| `--output-dir` | Yes | Hugo build output directory (e.g. `public/`) |
| `--redirects-kvs-name` | Yes | CloudFront KVS name for redirect data |
| `--headers-kvs-name` | Yes | CloudFront KVS name for header data |
| `--request-function-name` | Yes | CloudFront Function name for viewer-request |
| `--response-function-name` | Yes | CloudFront Function name for viewer-response |
| `--dry-run` | No | Parse and validate only, print plan, don't mutate AWS |
| `--config` | No | Path to config file (default: `hedgerules.toml` in current directory) |
| `--region` | No | AWS region override |

### `hedgerules validate`

Parse and validate only. Same flags as `deploy` minus the function/KVS names (only needs `--output-dir`). Exits 0 on success, 1 on validation failure.

```
hedgerules validate --output-dir public/
```

### `hedgerules version`

Print version and exit.

### Subcommand dispatch

Use Go's `os.Args` to dispatch:

```go
func main() {
    if len(os.Args) < 2 {
        usage()
        os.Exit(1)
    }
    switch os.Args[1] {
    case "deploy":
        runDeploy(os.Args[2:])
    case "validate":
        runValidate(os.Args[2:])
    case "version":
        fmt.Println(version)
    default:
        usage()
        os.Exit(1)
    }
}
```

---

## 3. Key Types and Interfaces

### Core data types

```go
// internal/kvs/types.go

// KVSEntry is a single key-value pair destined for CloudFront KVS.
type KVSEntry struct {
    Key   string
    Value string
}

// KVSData holds all entries for a single KVS, with validation methods.
type KVSData struct {
    Entries []KVSEntry
}

// SyncPlan describes what operations are needed to bring KVS to desired state.
type SyncPlan struct {
    Puts    []KVSEntry // Keys to add or update
    Deletes []string   // Keys to remove
}
```

### Validation

```go
// internal/kvs/validate.go

const (
    MaxKeyBytes       = 512
    MaxEntryBytes     = 1024      // key + value
    MaxTotalBytes     = 5_242_880 // 5 MB
)

// ValidationError describes a single constraint violation.
type ValidationError struct {
    Key     string
    Message string
}

// Validate checks all KVS constraints. Returns nil if valid.
func (d *KVSData) Validate() []ValidationError
```

### Hugo parsing

```go
// internal/hugo/directories.go

// ScanDirectories walks outputDir and returns index redirect entries
// (e.g., /blog -> /blog/).
func ScanDirectories(outputDir string) ([]kvs.KVSEntry, error)

// internal/hugo/redirects.go

// ParseRedirects reads the _redirects file and returns redirect entries.
func ParseRedirects(outputDir string) ([]kvs.KVSEntry, error)

// internal/hugo/headers.go

// ParseHeaders reads _cfheaders.json and returns header entries.
// Each entry's value is newline-delimited "Header-Name: value" strings.
func ParseHeaders(outputDir string) ([]kvs.KVSEntry, error)
```

### AWS operations

```go
// internal/kvs/sync.go

// ComputeSyncPlan compares desired state against existing KVS keys.
func ComputeSyncPlan(desired *KVSData, existingKeys map[string]string) *SyncPlan

// Sync applies a SyncPlan to a CloudFront KVS. Returns error on failure.
// Uses UpdateKeys batch API for efficiency.
func Sync(ctx context.Context, client KVSClient, kvsARN string, plan *SyncPlan) error

// internal/functions/deploy.go

// DeployFunction creates or updates a CloudFront Function with the given
// JS source code and KVS association. It publishes the function.
func DeployFunction(ctx context.Context, client CFClient, name string, code []byte, kvsARN string) error
```

### Interfaces for testability

```go
// internal/kvs/sync.go

// KVSClient abstracts the CloudFront KeyValueStore API for testing.
type KVSClient interface {
    DescribeKeyValueStore(ctx context.Context, params *cfkvs.DescribeKeyValueStoreInput, ...) (*cfkvs.DescribeKeyValueStoreOutput, error)
    ListKeys(ctx context.Context, params *cfkvs.ListKeysInput, ...) (*cfkvs.ListKeysOutput, error)
    UpdateKeys(ctx context.Context, params *cfkvs.UpdateKeysInput, ...) (*cfkvs.UpdateKeysOutput, error)
}

// internal/functions/deploy.go

// CFClient abstracts the CloudFront Functions API for testing.
type CFClient interface {
    DescribeFunction(ctx context.Context, params *cf.DescribeFunctionInput, ...) (*cf.DescribeFunctionOutput, error)
    CreateFunction(ctx context.Context, params *cf.CreateFunctionInput, ...) (*cf.CreateFunctionOutput, error)
    UpdateFunction(ctx context.Context, params *cf.UpdateFunctionInput, ...) (*cf.UpdateFunctionOutput, error)
    PublishFunction(ctx context.Context, params *cf.PublishFunctionInput, ...) (*cf.PublishFunctionOutput, error)
}
```

---

## 4. Data Flow

```
Hugo build output (public/)
      |
      v
+-----+------+--------+
|             |        |
v             v        v
Scan dirs   Parse    Parse
for index   _redirects  _cfheaders.json
redirects
|             |        |
v             v        v
[]KVSEntry  []KVSEntry  []KVSEntry
|             |        |
+------+------+        |
       |                |
       v                v
  Merge redirects    Header entries
  (file redirects    (separate KVS)
   override dirs)
       |                |
       v                v
    KVSData          KVSData
    (redirects)      (headers)
       |                |
       v                v
    Validate         Validate
       |                |
       +-------+--------+
               |
        (if --dry-run: print plan and exit)
               |
       +-------+--------+
       |                |
       v                v
  Fetch existing   Fetch existing
  redirect KVS     header KVS
  keys             keys
       |                |
       v                v
  ComputeSyncPlan  ComputeSyncPlan
       |                |
       v                v
  Sync to KVS      Sync to KVS
       |                |
       +-------+--------+
               |
               v
   Deploy viewer-request    Deploy viewer-response
   CloudFront Function      CloudFront Function
   (with redirect KVS ID)   (with header KVS ID)
```

### Merge precedence for redirects

Directory-derived index redirects (`/blog` -> `/blog/`) are generated first. Then `_redirects` file entries are applied on top, overriding any conflicts. This matches the existing Python behavior.

### Concurrency

The redirect KVS sync and header KVS sync are independent and can run concurrently. Similarly, the two function deployments are independent. Use `errgroup` for parallel execution with shared error handling:

```go
g, ctx := errgroup.WithContext(ctx)
g.Go(func() error { return syncRedirects(ctx, ...) })
g.Go(func() error { return syncHeaders(ctx, ...) })
if err := g.Wait(); err != nil { ... }
```

---

## 5. AWS API Interactions

### Services used

| Service | SDK Package | Operations |
|---|---|---|
| CloudFront | `github.com/aws/aws-sdk-go-v2/service/cloudfront` | `ListKeyValueStores`, `DescribeFunction`, `CreateFunction`, `UpdateFunction`, `PublishFunction` |
| CloudFront KVS | `github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore` | `DescribeKeyValueStore`, `ListKeys`, `UpdateKeys` |

### KVS sync strategy

Use `UpdateKeys` batch API (not individual `PutKey`/`DeleteKey`). This is an atomic operation that handles puts and deletes in a single call with ETag-based optimistic concurrency.

```go
// Simplified flow
resp, err := kvsClient.DescribeKeyValueStore(ctx, &cfkvs.DescribeKeyValueStoreInput{
    KvsARN: &kvsARN,
})
etag := resp.ETag

// ... compute puts and deletes ...

_, err = kvsClient.UpdateKeys(ctx, &cfkvs.UpdateKeysInput{
    KvsARN:  &kvsARN,
    IfMatch: etag,
    Puts:    puts,    // []types.PutKeyRequestListItem
    Deletes: deletes, // []types.DeleteKeyRequestListItem
})
```

This is a significant improvement over the Python scripts, which make individual `PutKey`/`DeleteKey` calls in a loop and must track ETag changes after each call.

### CloudFront Function deployment strategy

1. Call `DescribeFunction` to check if function exists.
2. If not found, call `CreateFunction` with the JS code and KVS association.
3. If found, call `UpdateFunction` with new code, using the returned ETag.
4. Call `PublishFunction` to make it live (from `DEVELOPMENT` to `LIVE` stage).

### KVS ARN resolution

Resolve KVS name to ARN by calling `ListKeyValueStores` and matching by name, same as the Python scripts. Cache the result for the duration of the command.

---

## 6. Embedding CloudFront Function JS Code

Use Go 1.16+ `embed` package to compile JS source files into the binary.

```go
// internal/functions/embed.go
package functions

import "embed"

//go:embed viewer-request.js
var ViewerRequestJS []byte

//go:embed viewer-response.js
var ViewerResponseJS []byte
```

The `functions/` directory at the project root holds the JS files. The `embed.go` file in `internal/functions/` references them via a relative path configured in `go:embed` directives.

**Note on the embed path**: Since `go:embed` paths are relative to the source file, the `internal/functions/` package will need access to the JS files. Two options:

- **Option A (preferred)**: Place the JS files in `internal/functions/` directly.
- **Option B**: Place JS in a top-level `functions/` dir and use a symlink or copy at build time.

**Recommendation**: Option A. Place JS files at `internal/functions/viewer-request.js` and `internal/functions/viewer-response.js`. This is simplest - no build steps, no symlinks.

### Prepending the KVS ID

The existing JS files expect `const kvsId = '...';` to be defined before the function code. At deploy time, the Go code prepends this constant:

```go
func BuildFunctionCode(jsSource []byte, kvsID string) []byte {
    header := fmt.Sprintf("const kvsId = '%s';\n", kvsID)
    return append([]byte(header), jsSource...)
}
```

---

## 7. Configuration Approach

### Config file format: TOML

Use TOML for the config file. It's the format Hugo uses, so Hedgerules users are already familiar with it.

File: `hedgerules.toml` (in project root, next to Hugo config).

```toml
# hedgerules.toml

output-dir = "public/"

[redirects]
kvs-name = "mysite-redirects"

[headers]
kvs-name = "mysite-headers"

[functions]
request-name = "mysite-viewer-request"
response-name = "mysite-viewer-response"
```

### Config resolution order (lowest to highest priority)

1. Defaults (none currently needed)
2. Config file (`hedgerules.toml`, or `--config` flag)
3. CLI flags

### Config file parsing

Use `github.com/BurntSushi/toml`. It's the standard Go TOML library, lightweight, no transitive dependencies.

### Config struct

```go
// internal/config/config.go

type Config struct {
    OutputDir string `toml:"output-dir"`
    Region    string `toml:"region"`
    DryRun    bool   // CLI-only, not in config file

    Redirects KVSConfig      `toml:"redirects"`
    Headers   KVSConfig      `toml:"headers"`
    Functions FunctionsConfig `toml:"functions"`
}

type KVSConfig struct {
    KVSName string `toml:"kvs-name"`
}

type FunctionsConfig struct {
    RequestName  string `toml:"request-name"`
    ResponseName string `toml:"response-name"`
}
```

---

## 8. Error Handling Strategy

### Principles

- **Fail fast**: Validate all data before making any AWS API calls. If validation fails, exit before mutating anything.
- **Wrap errors with context**: Use `fmt.Errorf("scanning directories in %s: %w", dir, err)` consistently.
- **No partial deploys on validation failure**: The `deploy` command validates both redirect and header data before syncing either.
- **Structured exit codes**: 0 = success, 1 = validation error, 2 = AWS API error, other = unexpected.

### Validation errors

Collect all validation errors before reporting, rather than stopping at the first one. This lets users fix all issues in one pass.

```go
errs := redirectData.Validate()
errs = append(errs, headerData.Validate()...)
if len(errs) > 0 {
    for _, e := range errs {
        fmt.Fprintf(os.Stderr, "validation error: %s: %s\n", e.Key, e.Message)
    }
    os.Exit(1)
}
```

### AWS API errors

- **ETag conflicts**: If `UpdateKeys` fails with `ConflictException`, report that another process modified the KVS concurrently. Do not retry automatically - the user should re-run.
- **Not found**: If KVS name doesn't resolve, print a clear message suggesting the user check the name and that the KVS exists.
- **Auth errors**: Let the AWS SDK error message pass through; don't wrap these excessively.

### Logging

Use `fmt.Fprintf(os.Stderr, ...)` for status/progress messages and `fmt.Println(...)` for structured output. No logging framework needed.

---

## 9. Dependency List

### Direct dependencies

| Module | Purpose |
|---|---|
| `github.com/aws/aws-sdk-go-v2` | AWS SDK core |
| `github.com/aws/aws-sdk-go-v2/config` | AWS SDK shared config loading |
| `github.com/aws/aws-sdk-go-v2/service/cloudfront` | CloudFront Functions + KVS listing API |
| `github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore` | CloudFront KVS data API |
| `github.com/BurntSushi/toml` | TOML config file parsing |
| `golang.org/x/sync` | `errgroup` for concurrent sync |

### Standard library only (no external dep needed)

| Need | stdlib solution |
|---|---|
| CLI flag parsing | `flag` |
| JSON parsing (`_cfheaders.json`) | `encoding/json` |
| File walking (directory scan) | `io/fs`, `filepath.WalkDir` |
| Embedding JS | `embed` |
| Concurrency | `context`, `sync` |
| Testing | `testing` |

### Go version

Require Go 1.21+ (for `slices`, `maps`, `slog` availability if needed later).

---

## Design Decisions Summary

| Decision | Choice | Rationale |
|---|---|---|
| CLI framework | stdlib `flag` | Small command surface, no need for cobra |
| AWS SDK | aws-sdk-go-v2 | Current/maintained SDK, required for CloudFront KVS |
| KVS sync | Batch `UpdateKeys` | Atomic, one API call vs N individual calls |
| Config format | TOML | Hugo users know it, lightweight parser |
| JS embedding | `go:embed` | No build step, compiled into binary |
| Concurrency | `errgroup` | Redirects + headers sync in parallel |
| Error strategy | Validate all first, fail fast | No partial deploys |
| Logging | stderr/stdout, no framework | Minimal dependencies |

---

## Relationship to Existing Examples

The existing `examples/micahrlweb/` directory demonstrates the full AWS setup with CloudFormation. Hedgerules replaces the Python scripts and the Jinja-templated CloudFormation function code deployment:

| Current (examples) | Hedgerules replacement |
|---|---|
| `kvs-request.py` | `hedgerules deploy` (redirect parsing + KVS sync) |
| `kvs-response.py` | `hedgerules deploy` (header parsing + KVS sync) |
| CFN template Jinja for JS + `const kvsId` | `internal/functions` embeds JS, prepends `kvsId`, deploys via API |
| Two separate scripts, manual KVS name args | One command, one config file |

The CloudFormation template in the example manages the Distribution, S3 buckets, and other infrastructure. Hedgerules intentionally does **not** manage these - it only manages the Functions and KVS data. Users create the Distribution and KVS resources out of band (via CloudFormation, Terraform, or console).

**Important**: Hedgerules **does** create/update CloudFront Functions, but expects the KVS to already exist. The KVS is typically created by the same infrastructure tool that creates the Distribution, since the Distribution's FunctionAssociations reference both the Function and the KVS.
