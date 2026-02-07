# Code Simplicity Recommendations

This document argues for doing LESS. Hedgerules is an open-source, non-monetizable project. Every line of code is a maintenance burden with zero revenue to offset it. The goal is the smallest possible codebase that solves the core problem.

## The Core Problem (keep this laser-focused)

The tool exists to do ONE thing: **sync data from a Hugo build output into CloudFront Key Value Stores**. That's it. Two KVS stores (redirects, headers), populated from files Hugo already produces. Everything else is scope creep.

---

## CUT: Features to Drop Entirely

### 1. Redirect chain following (CUT)

The viewer-request JS currently follows redirect chains up to 10 levels deep. This is complexity defending against a scenario that shouldn't exist: if a user has `/a -> /b` and `/b -> /c`, the tool should resolve this at sync time and write `/a -> /c` directly. Or better: just don't. Let the user manage their own redirect targets. If they write `/a -> /b` and `/b` itself redirects, the browser follows two 301s. This has worked on the web for 30 years.

- **Who needs this?** Almost nobody. Redirect chains are a smell, not a feature.
- **Maintenance cost:** Loop protection logic, max-redirect constants, harder-to-debug behavior.
- **Recommendation:** Single KVS lookup. One redirect per request. Done.

### 2. `{/path}` token substitution (CUT)

The response JS replaces `{/path}` in header values with the request path. The only use case in the examples is `Onion-Location`. This is a templating system baked into a CloudFront Function for one header on one site.

- **Who needs this?** Onion-Location users only, and they can hardcode the pattern or use a different mechanism.
- **Maintenance cost:** String replacement logic in a latency-critical path (CloudFront Functions have a 1ms budget). Template syntax that must be documented and explained.
- **Recommendation:** Drop it. If a user needs dynamic header values, they can write their own CloudFront Function. Hedgerules should set static headers only.

### 3. Extension wildcard matching (`*.xml`, `*.html`) (CUT)

The hierarchical header matching supports glob-style extension patterns. This is cute but adds branching complexity to the JS function and is rarely needed.

- **Who needs this?** Someone who wants different headers on all `.xml` files vs all `.html` files. This is a niche use case.
- **Maintenance cost:** Pattern parsing, ordering semantics, documentation of specificity rules.
- **Recommendation:** Drop it. Support exact paths and directory prefixes only. If someone needs headers on all XML files, they can enumerate the paths in their Hugo template.

### 4. Hierarchical/cascading header pattern matching (SIMPLIFY drastically)

The current system builds a cascade: `/` -> `/blog/` -> `/blog/post/` -> `*.html` -> `/blog/post/article.html`, with more-specific patterns overriding less-specific ones. This is a mini-CSS-specificity engine running in a CloudFront Function with a 1ms execution budget.

- **Who needs this?** Users who want "all pages get header X, but /blog/ pages also get header Y." In practice, most Hugo sites have 2-3 header rules total.
- **Maintenance cost:** HIGH. The specificity logic, pattern building, merge semantics, size tracking, and truncation logic account for ~80% of the response function.
- **Recommendation:** Support only exact-path header lookups. One KVS lookup per request. If the key exists, apply those headers. If not, done. Users who want headers on all pages put them in a Hugo template that generates the header config for every page. Hugo is already doing the site generation -- let it do the work, not the edge function.

### 5. CloudFormation template management (CUT from Hedgerules scope)

The CFN template in the example is 600 lines covering S3 buckets, CloudFront distributions, Route53, ACM certs, IAM groups, Athena, and Glue. Hedgerules should not touch any of this.

- **Who needs this?** Everyone, but they all need it differently (different domain setup, different logging preferences, different IAM models).
- **Maintenance cost:** Enormous. CFN is notoriously brittle. If Hedgerules ships a "create my infra" command, it owns every bug in every user's AWS account.
- **Recommendation:** Ship the CFN example as documentation only. Hedgerules takes a KVS ARN or name as input and syncs data. Period. The user creates their own infrastructure however they want (CFN, CDK, Terraform, console clicks).

### 6. CloudFront Function management/deployment (CUT from Hedgerules scope)

The README mentions `hedgerules deploy` creating the CloudFront Functions. Don't. The JS functions are ~30 lines each. They change almost never. Deploying them is a one-time setup, not a repeated operation.

- **Who needs this?** First-time setup only.
- **Maintenance cost:** CloudFront Function API calls, versioning, publish logic, association management.
- **Recommendation:** Ship the two JS files as static assets in the repo. Document how to deploy them manually or via CFN/CDK. Hedgerules only syncs KVS data.

### 7. Dry-run mode (CUT)

Both Python scripts have `--dry-run`. It just prints what would be uploaded. The tool already prints a summary before uploading.

- **Who needs this?** Paranoid users. But the data is in KVS, not a database -- it's trivially reversible by re-running the tool.
- **Maintenance cost:** Low, but it's one more code path to maintain and test.
- **Recommendation:** Drop it. The tool prints what it will do, then does it. If the user wants to inspect first, they can read the generated JSON files directly.

### 8. Debug headers (`x-mrldbg-*`) (CUT)

The response function adds multiple debug headers to every response: matched patterns, size tracking, truncation status. These are specific to the example author's debugging needs.

- **Who needs this?** The original developer, during development. Nobody else.
- **Maintenance cost:** Size budget consumed on every response. Information leakage in production.
- **Recommendation:** Remove from the shipped JS function. If users want debug headers, they can add them to their own fork.

### 9. Multi-cloud support (DON'T BUILD)

The README mentions "future ideas" for Azure and GCP. No.

- **Who needs this?** A different project's users.
- **Maintenance cost:** 3x the API surface, 3x the testing, 3x the auth models.
- **Recommendation:** This is a CloudFront tool. Name it that. Own that. If someone wants Azure, they fork it or build a different tool.

---

## KEEP: The Essential Core

### 1. Scan Hugo output for directory paths -> generate redirect KVS entries (KEEP)

This is the core value: `/path` -> `/path/` for every directory. One function, one loop, simple output.

### 2. Parse `_redirects` file -> merge into redirect KVS entries (KEEP)

Hugo generates `_redirects` from aliases. Reading this file and merging is straightforward and essential.

### 3. Parse `_cfheaders.json` -> generate header KVS entries (KEEP, but simplify)

Headers are useful. But simplify the format: exact paths only, no pattern matching (see cut #4 above). The JSON maps paths to header strings. Simple.

### 4. KVS size validation (KEEP)

CloudFront KVS has hard limits. Validating before upload avoids confusing API errors. This is a few lines of code and high value.

### 5. KVS sync: diff current state, add/update/delete (KEEP, but simplify)

The sync-with-diff approach is correct. But simplify: use the `UpdateKeys` batch API instead of individual put/delete calls. One API call instead of N.

### 6. Viewer request function: single KVS lookup + index.html rewrite (KEEP)

After cutting chain following: look up URI in KVS. If found, 301 redirect. If URI ends in `/`, append `index.html`. Return. This is ~15 lines of JS.

### 7. Viewer response function: exact-path header lookup (KEEP, simplified)

After cutting hierarchical matching: look up URI in KVS. If found, parse and apply headers. Return. This is ~15 lines of JS.

---

## SIMPLIFY: Architecture and Implementation

### Command structure: ONE command, no subcommands

`hedgerules sync --kvs-request <name> --kvs-response <name> <hugo-output-dir>`

That's it. One command. It scans the directory, builds both KVS datasets, validates, and syncs both. No `deploy`, no `init`, no `validate` subcommand. One command that does the one thing.

Stretch consideration: even the flags could be eliminated if we adopt a convention like looking for a `hedgerules.toml` in the Hugo root. But honestly, three CLI flags is fine. Don't build a config file parser for three values.

### No config file

Three values are needed: request KVS name, response KVS name, and the Hugo output directory. These fit in CLI flags. A config file is overhead for three strings.

If there's pressure for a config file, use Hugo's own config (`hugo.toml` under `[params.hedgerules]`). Don't invent a new config format.

### Dependencies: minimal

- Standard library `flag` package, not cobra/viper/etc. Three flags don't need a framework.
- AWS SDK for Go v2 (unavoidable, but use only `cloudfront` and `cloudfront-keyvaluestore` clients).
- No other dependencies.

### Code structure: flat

```
main.go          # CLI parsing, orchestration
kvs.go           # KVS sync logic (shared between request/response)
scan.go          # Hugo output scanning (directories, _redirects, _cfheaders.json)
validate.go      # KVS size constraint validation
```

Four files. No `internal/` directory. No `pkg/`. No interfaces unless there are two implementations. No `cmd/` directory -- there's one command.

### Error handling: fail fast, fail loud

- No retries. If an API call fails, print the error and exit 1.
- No partial syncs. Build the full dataset, validate, then sync. If validation fails, sync nothing.
- No graceful degradation. If `_redirects` is malformed, error and stop. Don't skip bad lines.
- No recovery. If sync fails midway, the user re-runs the tool. KVS is convergent -- re-running is safe.

### Testing: test the data, not the infra

- Test directory scanning produces correct redirect maps.
- Test `_redirects` parsing produces correct redirect maps.
- Test `_cfheaders.json` parsing produces correct header maps.
- Test KVS validation catches size violations.
- Test merge logic (file redirects override directory redirects).
- Do NOT write integration tests against real AWS. Do NOT mock the AWS SDK. Test the pure data transformation functions only.

### AWS API surface: minimum

Only these API calls:

1. `cloudfront:ListKeyValueStores` - resolve KVS name to ARN
2. `cloudfront-keyvaluestore:DescribeKeyValueStore` - get current ETag
3. `cloudfront-keyvaluestore:ListKeys` - get current keys for diff
4. `cloudfront-keyvaluestore:UpdateKeys` - batch put/delete in one call (NOT individual `PutKey`/`DeleteKey`)

Four API calls total. The Python examples use individual put/delete which is N calls for N keys. The `UpdateKeys` batch API does it in one.

---

## Summary: Lines of Code Budget

| Component | Estimated LoC | Notes |
|---|---|---|
| `main.go` | ~50 | Flag parsing, call scan, validate, sync |
| `scan.go` | ~80 | Directory walk, `_redirects` parse, `_cfheaders.json` parse |
| `kvs.go` | ~60 | List current keys, diff, batch UpdateKeys |
| `validate.go` | ~30 | Size constraint checks |
| `viewer-request.js` | ~15 | Single lookup + index.html rewrite |
| `viewer-response.js` | ~15 | Single lookup + apply headers |
| **Total** | **~250** | |

The Python examples are ~575 lines for just the sync logic (not counting JS or CFN). The Go version should be under 250 lines of actual logic, plus whatever the AWS SDK boilerplate requires.

---

## What Happens If We Just Don't?

For every feature questioned above, I asked "what happens if we just don't do this?"

| Feature | What happens without it |
|---|---|
| Redirect chains | Browser follows two 301s. Works fine. Always has. |
| `{/path}` token | User hardcodes their onion URL or writes a custom function. |
| Extension wildcards | User enumerates paths in Hugo template. Hugo is good at this. |
| Hierarchical headers | User generates per-page headers in Hugo. Hugo is good at this too. |
| CFN management | User deploys infra their way. They already do this for S3/CloudFront. |
| Function deployment | User copies 15 lines of JS once. |
| Dry run | User reads the JSON file before running sync. |
| Debug headers | User adds them to their own function if needed. |
| Multi-cloud | Someone else builds a different tool. |

None of these are catastrophic. The tool still solves the core problem: getting Hugo's redirect and header data into CloudFront KVS.
