# Hedgerules: Design Decisions

Resolving conflicts between [UX advocate]({{< relref "ux-recommendations" >}}) and [minimalist]({{< relref "minimalist-recommendations" >}}), accounting for the fact that this is an open-source, non-monetizable project.

**Decision framework**: Enough UX to get someone started, minimal code to maintain. Every feature must justify its maintenance cost with zero revenue to offset it.

---

## Feature Decisions

### KEEP: Core

| Feature | Decision | Rationale |
|---|---|---|
| Scan dirs for index redirects | KEEP | Core value proposition |
| Parse `_redirects` | KEEP | Essential for Hugo aliases |
| Parse `_hedge_headers.json` | KEEP | Essential for custom headers |
| KVS constraint validation | KEEP | Prevents confusing API errors, ~30 lines |
| KVS batch sync (`UpdateKeys`) | KEEP | The whole point of the tool |
| Dry-run mode | KEEP | ~10 lines, huge confidence boost for new users |
| CloudFront Function deployment | KEEP | Part of the core value prop; users need KVS ID injected into JS |
| Config file (`hedgerules.toml`) | KEEP | Goes in version control, teammates don't memorize flags |
| Redirect chain following | KEEP | Follows redirect chains to final destination at deploy time |
| `{/path}` token substitution | KEEP | Allows path-based token replacement in header values |
| Extension wildcard matching (`*.xml`) | KEEP | Match headers by file extension |
| Full hierarchical header cascade | KEEP | Root → directory → extension → exact path, with specificity rules |
| Debug headers (`x-hedgerules-*`) | KEEP | Off by default; enabled via variable injection (same mechanism as KVS name) |

### CUT: Not building

| Feature | Decision | Rationale |
|---|---|---|
| `hedgerules init` command | CUT | Generates a 5-line config. Users copy from docs. |
| `hedgerules check` command | CUT | Deploy fails with clear errors. Separate check is redundant. |
| `hedgerules validate` command | CUT | `deploy --dry-run` covers this. |
| STS credential pre-check | CUT | Just let it fail. AWS SDK errors are clear enough. |
| Multi-cloud (Azure, GCP) | CUT | Different project. This is a CloudFront tool. |
| CloudFormation management | CUT | Too variable per user. Ship example as docs only. |
| Auto-discovery of KVS by convention | CUT | Magic that breaks. Users specify names explicitly. |
| Retry logic / partial sync recovery | CUT | Re-run is safe. KVS is convergent. |

### Header matching

Full hierarchical header cascade: root `/` → directory → extension → exact path, with specificity rules. More specific matches take priority over less specific ones.

- Root `/` match: applied as defaults to all requests
- Directory match: `/blog/` headers apply to all paths under `/blog/`
- Extension match: `*.xml` headers apply to all `.xml` files
- Exact path match: `/blog/my-post/` → apply those headers

More specific matches override less specific ones. Extension and exact-path headers take priority over directory headers, which take priority over root headers.

Hugo generates `_hedge_headers.json` with per-page header data. The CloudFront Function performs hierarchical lookups at the edge.

---

## Architecture Decisions

### CLI structure

```
hedgerules deploy [flags]    # Primary: sync KVS + deploy functions
hedgerules version           # Print version
```

Two commands. `deploy --dry-run` replaces `validate`.

### Package layout

```
cmd/hedgerules/main.go       # Entry point, flag parsing, config loading
internal/hugo/               # Parse directories, _redirects, _hedge_headers.json
internal/kvs/                # Validation, diff, sync via UpdateKeys
internal/functions/          # Embed JS, deploy CloudFront Functions
  viewer-request.js          # Embedded: single-lookup redirect + index rewrite
  viewer-response.js         # Embedded: exact-path + root fallback headers
go.mod
hedgerules.toml              # Example config (in repo root for documentation)
```

Keep `internal/` — it's Go-idiomatic. Drop `internal/config/` (config loading goes in main) and `internal/aws/` (client construction goes in main). Three internal packages.

### Config file

```toml
# hedgerules.toml
output-dir = "public"

[redirects]
kvs-name = "mysite-redirects"

[headers]
kvs-name = "mysite-headers"

[functions]
request-name = "mysite-viewer-request"
response-name = "mysite-viewer-response"
```

CLI flags override config. Config is optional (all values can come from flags). But recommended: it goes in version control.

### Dependencies

| Dependency | Justification |
|---|---|
| aws-sdk-go-v2 + cloudfront + cloudfrontkeyvaluestore | Unavoidable |
| BurntSushi/toml | Config file parsing, Hugo users know TOML |
| stdlib flag | CLI parsing |
| stdlib embed | JS bundling |

**Drop**: `golang.org/x/sync` (errgroup). Sequential sync is fine — two API calls don't benefit meaningfully from concurrency. Less code, simpler error handling.

### Error handling

- Validate ALL data before ANY AWS call
- Collect all validation errors, report together
- Fail fast on AWS errors, no retries
- Clear messages: what happened, what to do
- Exit 0 = success, exit 1 = error

### Testing

Test pure data transformation functions only:
- Directory scanning → redirect entries
- `_redirects` parsing → redirect entries
- `_hedge_headers.json` parsing → header entries
- Merge logic (file redirects override directory redirects)
- Validation (key size, entry size, total size)

No AWS mocks. No interface abstractions for testability. Test the logic, not the plumbing.

---

## CloudFront Functions

### viewer-request.js

Single KVS lookup. If found, 301 redirect. Follows redirect chains at deploy time (resolved before writing to KVS). If URI ends in `/`, append `index.html`. Done.

### viewer-response.js

Hierarchical KVS lookups: exact path, extension, directory, then root `/`. Parse newline-delimited `Header: value` strings. More specific matches override less specific ones. Supports `{/path}` token substitution in header values.

Debug headers (`x-hedgerules-*`) are available but off by default. Enabled via variable injection into the function code at deploy time, using the same mechanism that injects the KVS name.
