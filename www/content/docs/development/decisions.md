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
| Parse `_cfheaders.json` | KEEP | Essential for custom headers |
| KVS constraint validation | KEEP | Prevents confusing API errors, ~30 lines |
| KVS batch sync (`UpdateKeys`) | KEEP | The whole point of the tool |
| Dry-run mode | KEEP | ~10 lines, huge confidence boost for new users |
| CloudFront Function deployment | KEEP | Part of the core value prop; users need KVS ID injected into JS |
| Config file (`hedgerules.toml`) | KEEP | Goes in version control, teammates don't memorize flags |

### CUT: Not building

| Feature | Decision | Rationale |
|---|---|---|
| Redirect chain following | CUT | Browser follows two 301s. Works fine. |
| `{/path}` token substitution | CUT | One header on one site. Users write custom JS if needed. |
| Extension wildcard matching (`*.xml`) | CUT | Niche. Users enumerate paths in Hugo templates. |
| Full hierarchical header cascade | CUT | Mini-CSS-specificity engine in 1ms budget. Way too much. |
| `hedgerules init` command | CUT | Generates a 5-line config. Users copy from docs. |
| `hedgerules check` command | CUT | Deploy fails with clear errors. Separate check is redundant. |
| `hedgerules validate` command | CUT | `deploy --dry-run` covers this. |
| STS credential pre-check | CUT | Just let it fail. AWS SDK errors are clear enough. |
| Debug headers (`x-mrldbg-*`) | CUT | Author's debugging needs, not shipped feature. |
| Multi-cloud (Azure, GCP) | CUT | Different project. This is a CloudFront tool. |
| CloudFormation management | CUT | Too variable per user. Ship example as docs only. |
| Auto-discovery of KVS by convention | CUT | Magic that breaks. Users specify names explicitly. |
| Retry logic / partial sync recovery | CUT | Re-run is safe. KVS is convergent. |

### SIMPLIFIED: Header matching

**Instead of**: Full hierarchical cascade (root → directory → extension → exact path, with specificity rules, size tracking, truncation logic — ~80% of the response function)

**Do this**: Exact-path lookup + root `/` fallback. Two KVS lookups maximum per request.

- Exact path match: `/blog/my-post/` → apply those headers
- Root `/` match: apply as defaults (exact path headers take priority)
- No wildcards, no directory hierarchy, no extension patterns

This handles 95% of use cases (global security headers + per-page overrides) in ~20 lines of JS.

**For users who want the same headers on every page**: Hugo generates `_cfheaders.json` with the header on every page path. Hugo is already enumerating pages — let it do the work.

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
internal/hugo/               # Parse directories, _redirects, _cfheaders.json
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
- `_cfheaders.json` parsing → header entries
- Merge logic (file redirects override directory redirects)
- Validation (key size, entry size, total size)

No AWS mocks. No interface abstractions for testability. Test the logic, not the plumbing.

---

## Simplified CloudFront Functions

### viewer-request.js (~15 lines)

Single KVS lookup. If found, 301 redirect. If URI ends in `/`, append `index.html`. Done.

No chain following. No loop. No max-redirect constant.

### viewer-response.js (~25 lines)

Two KVS lookups: exact path, then root `/`. Parse newline-delimited `Header: value` strings. Exact path headers win over root. Done.

No hierarchical cascade. No wildcards. No `{/path}` token. No size tracking. No debug headers.
