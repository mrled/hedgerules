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
      main.go              # Entry point, CLI flags, command dispatch, orchestration
  internal/
    hugo/
      directories.go       # Scan Hugo output dirs for index redirects
      redirects.go         # Parse _hedge_redirects.txt, merge redirects
      headers.go           # Parse _hedge_headers.json
    kvs/
      types.go             # Entry, Data, SyncPlan types
      validate.go          # KVS constraint validation
      sync.go              # Diff + sync logic (put/delete)
    functions/
      embed.go             # go:embed for JS function code, BuildFunctionCode
      deploy.go            # Create/update CloudFront Functions via API
      viewer-request.js    # CloudFront Function: redirects + index rewrite
      viewer-response.js   # CloudFront Function: custom response headers (cascade)
  hedgerules.toml          # Example config file
  go.mod
  go.sum
```

### Package responsibilities

| Package | Responsibility |
|---|---|
| `cmd/hedgerules` | CLI entry point, flag parsing, TOML config loading, command dispatch |
| `internal/hugo` | Parse Hugo build output: directories, `_hedge_redirects.txt`, `_hedge_headers.json`; merge redirects |
| `internal/kvs` | KVS data types, validation against constraints, diff-and-sync to AWS |
| `internal/functions` | Embed JS source files, inject variables (`kvsId`, `debugHeaders`), deploy to CloudFront Functions API |

---

## 2. CLI Command Structure

Use **stdlib `flag`** package only. No third-party CLI framework. The tool has a small surface area (one primary command) and doesn't justify a dependency like cobra.

```
hedgerules deploy [flags]
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

// Entry is a single key-value pair destined for CloudFront KVS.
type Entry struct {
    Key   string
    Value string
}

// Data holds all entries for a single KVS, with validation methods.
type Data struct {
    Entries []Entry
}

// SyncPlan describes what operations are needed to bring KVS to desired state.
type SyncPlan struct {
    Puts    []Entry  // Keys to add or update
    Deletes []string // Keys to remove
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
func ScanDirectories(outputDir string) ([]kvs.Entry, error)

// internal/hugo/redirects.go

// ParseRedirects reads _hedge_redirects.txt and returns redirect entries.
func ParseRedirects(outputDir string) ([]kvs.Entry, error)

// MergeRedirects merges directory redirects with file redirects.
// File redirects take precedence.
func MergeRedirects(dirEntries, fileEntries []kvs.Entry) []kvs.Entry

// internal/hugo/headers.go

// ParseHeaders reads _hedge_headers.json and returns header entries.
// Each entry's value is newline-delimited "Header-Name: value" strings.
func ParseHeaders(outputDir string) ([]kvs.Entry, error)
```

### AWS operations

```go
// internal/kvs/sync.go

// ComputeSyncPlan compares desired state against existing KVS keys.
func ComputeSyncPlan(desired *Data, existingKeys map[string]string) *SyncPlan

// Sync applies a SyncPlan to a CloudFront KVS. Returns error on failure.
// Uses UpdateKeys batch API for efficiency.
func Sync(ctx context.Context, client KVSClient, kvsARN, etag string, plan *SyncPlan) error

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
for index   _hedge_     _hedge_headers.json
redirects   redirects.txt
|             |        |
v             v        v
[]Entry     []Entry    []Entry
|             |        |
+------+------+        |
       |                |
       v                |
  Merge redirects       |
  (file overrides dirs) |
       |                |
       v                v
    Data             Data
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
   (with redirect KVS ID)   (with header KVS ID +
                              debugHeaders flag)
```

### Merge precedence for redirects

Directory-derived index redirects (`/blog` -> `/blog/`) are generated first. Then `_redirects` file entries are applied on top, overriding any conflicts. This matches the existing Python behavior.

### Sequential execution

The redirect KVS sync and header KVS sync run sequentially (sync redirects, then sync headers, then deploy functions). This keeps the code simple and avoids error-handling complexity from concurrency.

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

The JS files live in `internal/functions/` alongside `embed.go`. Since `go:embed` paths are relative to the source file, this is the simplest arrangement — no build steps, no symlinks.

### Injecting variables

At deploy time, the Go code prepends runtime variables to the JS source:

```go
func BuildFunctionCode(jsSource []byte, kvsID string, debugHeaders bool) []byte {
    header := fmt.Sprintf("var kvsId = '%s';\nvar debugHeaders = %v;\n", kvsID, debugHeaders)
    return append([]byte(header), jsSource...)
}
```

This injects:
- `kvsId` — the CloudFront KVS ARN for KVS lookups
- `debugHeaders` — whether to emit `x-hedgerules-*` debug response headers (off by default)

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
# debug-headers = false
```

### Config resolution order (lowest to highest priority)

1. Defaults (none currently needed)
2. Config file (`hedgerules.toml`, or `--config` flag)
3. CLI flags

### Config file parsing

Use `github.com/BurntSushi/toml`. It's the standard Go TOML library, lightweight, no transitive dependencies.

### Config struct

```go
// cmd/hedgerules/main.go

type config struct {
    OutputDir string `toml:"output-dir"`
    Region    string `toml:"region"`

    Redirects kvsConfig       `toml:"redirects"`
    Headers   kvsConfig       `toml:"headers"`
    Functions functionsConfig `toml:"functions"`
}

type kvsConfig struct {
    KVSName string `toml:"kvs-name"`
}

type functionsConfig struct {
    RequestName  string `toml:"request-name"`
    ResponseName string `toml:"response-name"`
    DebugHeaders bool   `toml:"debug-headers"`
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

### Standard library only (no external dep needed)

| Need | stdlib solution |
|---|---|
| CLI flag parsing | `flag` |
| JSON parsing (`_hedge_headers.json`) | `encoding/json` |
| File walking (directory scan) | `io/fs`, `filepath.WalkDir` |
| Embedding JS | `embed` |
| Context | `context` |
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
| Concurrency | Sequential | Simple, no parallel complexity |
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

---

## Updates 20260208

The following changes were decided in the [Design Decisions]({{< relref "decisions" >}}) doc. Each item needs to be reflected across the codebase (Go code, JS functions, Hugo theme, docs, and tests).

### Terminology rename

- `_hedge_headers.json` is now `_hedge_headers.json`. All references in code, config, docs, templates, and tests must be updated.

### Features re-added (previously cut)

1. **~~Redirect chain following~~** — Remains cut. The browser follows multiple 301s natively; chain resolution is unnecessary complexity.
2. **`{/path}` token substitution** — Support `{/path}` tokens in header values. The viewer-response function substitutes the request path into header values at the edge.
3. **Extension wildcard matching (`*.xml`)** — Match headers by file extension in addition to exact path and directory. The viewer-response function needs an extension-based KVS lookup.
4. **Full hierarchical header cascade** — Replace the simplified exact-path + root fallback with a full cascade: root `/` → directory → extension → exact path. More specific matches override less specific ones. This affects both the Go CLI (how header entries are organized into KVS) and viewer-response.js (lookup order).

### Debug headers

- **Re-added** as `x-hedgerules-*` (previously `x-mrldbg-*`, previously cut entirely).
- **Off by default.** Enabled via variable injection into the CloudFront Function code at deploy time, using the same `BuildFunctionCode` mechanism that already injects the KVS name.
- The Go CLI needs a config/flag to toggle debug header injection. The viewer-response.js function needs conditional debug header logic gated on the injected variable.
