# Hedgerules UX Recommendations

Prioritized recommendations for making Hedgerules simple and frictionless for users. Each recommendation includes rationale and specific implementation guidance.

---

## Priority 1: Zero-Config First Run (Critical)

### 1.1 Auto-detect Hugo output directory

**Problem:** The Python scripts require the user to pass `output_dir` as a positional argument (e.g., `public/prod-obverse`). Hugo's output directory varies: it defaults to `public/`, but `hugo.toml` can override it via `publishDir`. Multi-environment setups (like the micahrlweb example) use subdirectories.

**Recommendation:** Hedgerules should auto-detect the Hugo output directory by:
1. Looking for `hugo.toml` / `hugo.yaml` / `hugo.json` in the current directory (or a `--project` flag).
2. Reading the `publishDir` config value if set.
3. Falling back to `public/` if no config found.
4. Only require the user to specify the directory when auto-detection fails.

This means `hedgerules deploy` with zero arguments should just work for most Hugo sites.

### 1.2 Auto-discover KVS names from CloudFormation/config

**Problem:** The current scripts require `--kvs-name` for both the request KVS (redirects) and response KVS (headers). Users must look up KVS names from CloudFormation outputs or the AWS console. This is two separate values they need to find and pass correctly.

**Recommendation:** Support multiple discovery strategies, tried in order:
1. **Config file** (`hedgerules.toml` or a `[hedgerules]` section in `hugo.toml`) — the user sets it once and it sticks.
2. **Convention-based naming** — if the user gives a `stack-name`, derive KVS names as `{stack-name}-redirects` and `{stack-name}-headers` (matching the CloudFormation template convention).
3. **AWS auto-discovery** — list KVS stores and look for ones with `redirects`/`headers` suffixes.
4. **Explicit flags** — `--request-kvs` and `--response-kvs` as escape hatches.

The goal: most users configure once in their `hedgerules.toml` and never think about KVS names again.

### 1.3 Single config file with sensible defaults

**Problem:** Users currently need to know about multiple moving parts: Hugo output dir, KVS names for requests, KVS names for responses, AWS region, CloudFront distribution ID, etc.

**Recommendation:** Provide a single `hedgerules.toml` config file. Minimal viable config:

```toml
# hedgerules.toml — minimal config
stack-name = "my-site"
```

That's it. Everything else derived by convention. Full config for power users:

```toml
# hedgerules.toml — full config
stack-name = "my-site"
hugo-output-dir = "public/"
request-kvs = "my-site-redirects"
response-kvs = "my-site-headers"
region = "us-east-1"
```

---

## Priority 2: One-Step Operations (High)

### 2.1 `hedgerules deploy` does everything

**Problem:** The current workflow requires running two separate Python scripts (`kvs-request.py` and `kvs-response.py`), each with their own arguments. This is two commands with four flags the user must get right.

**Recommendation:** `hedgerules deploy` should:
1. Scan the Hugo output directory for directories (to generate index redirects).
2. Read `_redirects` for Hugo alias redirects.
3. Read `_cfheaders.json` for custom headers.
4. Validate all KVS constraints.
5. Sync redirects to the request KVS.
6. Sync headers to the response KVS.
7. Print a summary.

One command. No arguments needed if config file exists or conventions are followed.

### 2.2 `hedgerules deploy --dry-run` for safe previews

**Problem:** Users deploying for the first time are nervous about pushing to production. The current scripts have `--dry-run`, but it's per-script.

**Recommendation:** A single `hedgerules deploy --dry-run` that shows the full sync plan for both KVS stores:
```
Redirects (request KVS: my-site-redirects):
  Add:    47 keys
  Update:  3 keys
  Delete:  1 key

Headers (response KVS: my-site-headers):
  Add:    12 keys
  Update:  2 keys
  Delete:  0 keys

Run without --dry-run to apply.
```

This gives users confidence before their first real deploy.

### 2.3 `hedgerules init` for first-time setup

**Problem:** New users don't know what config they need, what format headers should be in, or how their Hugo site should be structured to work with Hedgerules.

**Recommendation:** `hedgerules init` should:
1. Check if we're in a Hugo project directory (look for `hugo.toml`).
2. Ask the user for their CloudFormation stack name (or KVS names if not using CFN).
3. Generate a `hedgerules.toml` with comments explaining each field.
4. Optionally create a starter `_cfheaders.json` template (Hugo template file) if the user wants custom headers.
5. Print next steps: "Run `hugo build` then `hedgerules deploy --dry-run` to preview."

Keep `init` non-interactive if possible — accept arguments on the command line and only prompt when truly needed.

---

## Priority 3: Clear Errors and Guidance (High)

### 3.1 Actionable error messages with next steps

**Problem:** AWS errors are notoriously cryptic. "AccessDenied" could mean wrong IAM permissions, wrong region, wrong account, or a non-existent resource. The current Python scripts just print raw exceptions.

**Recommendation:** Wrap every AWS interaction with user-friendly error handling. Specific cases:

| Error | User-facing message |
|-------|-------------------|
| KVS not found | `Error: Key Value Store "my-site-redirects" not found. Check your stack-name in hedgerules.toml, or verify the KVS exists with: aws cloudfront list-key-value-stores` |
| Access denied | `Error: Permission denied when accessing KVS. Ensure your AWS credentials have cloudfront-keyvaluestore:* permissions. See: hedgerules.example.com/docs/aws-permissions` |
| No Hugo output | `Error: Hugo output directory "public/" not found. Did you run "hugo" first?` |
| No _redirects | `Warning: No _redirects file found in public/. No Hugo alias redirects will be synced. (This is fine if you don't use Hugo aliases.)` |
| No _cfheaders.json | `Warning: No _cfheaders.json found in public/. No custom headers will be synced. (This is fine if you don't need custom response headers.)` |
| KVS key too large | `Error: Redirect "/very/long/path/that/exceeds/the/limit/..." exceeds the 512-byte key limit. CloudFront KVS has a 512-byte key limit. Consider shorter URL paths.` |
| KVS total too large | `Error: Total data (6.2 MB) exceeds the 5 MB CloudFront KVS limit. You have 4,312 entries. Consider reducing the number of redirects or header entries.` |
| Partial sync failure | `Warning: 3 of 47 keys failed to sync. Run "hedgerules deploy" again to retry. Failed keys: /path/a, /path/b, /path/c` |

Every error should tell the user: **what happened, why, and what to do next.**

### 3.2 Pre-flight validation before any AWS calls

**Problem:** The current scripts discover problems mid-sync (e.g., a key is too large) after already making some AWS calls.

**Recommendation:** Before touching AWS, validate everything locally:
1. Check Hugo output directory exists and isn't empty.
2. Parse all redirects and headers.
3. Validate all KVS constraints (key sizes, total size).
4. Report all problems at once (don't fail on the first one).
5. Only after all validation passes, start the AWS sync.

This saves users from partial syncs and round-trip latency when there's a local problem.

### 3.3 Check AWS credentials early

**Problem:** Users might have expired credentials, wrong profiles, or no credentials at all. Finding out after 30 seconds of processing is frustrating.

**Recommendation:** At the start of any AWS-touching command, do a cheap credential check (like `sts:GetCallerIdentity`). If it fails, print:
```
Error: AWS credentials not configured or expired.
Ensure you have valid credentials via environment variables, ~/.aws/credentials, or SSO.
Current profile: default
Run "aws sts get-caller-identity" to debug.
```

---

## Priority 4: Progressive Disclosure (Medium)

### 4.1 Simple things simple, complex things possible

**Problem:** The header system supports powerful features (hierarchical path matching, `{/path}` token substitution, extension wildcards). But most users just want basic security headers on all pages.

**Recommendation:** Document and support a layered complexity model:

**Level 1 — Basic (most users):** "Add security headers to your whole site"
```json
{
  "/": {
    "X-Frame-Options": "DENY",
    "X-Content-Type-Options": "nosniff"
  }
}
```

**Level 2 — Per-section:** "Different headers for your blog vs. your API docs"
```json
{
  "/": { "X-Frame-Options": "DENY" },
  "/blog/": { "Cache-Control": "public, max-age=3600" }
}
```

**Level 3 — Advanced:** "Extension wildcards and path tokens"
```json
{
  "*.xml": { "Content-Type": "application/xml; charset=utf-8" },
  "/": { "Onion-Location": "http://example.onion{/path}" }
}
```

The CLI output should reflect this: show simple summaries by default, and `--verbose` for the full pattern-matching details.

### 4.2 Don't require headers if the user doesn't need them

**Problem:** Not every Hugo site needs custom response headers. Users who just want redirects and index.html rewrites shouldn't need to deal with the header system at all.

**Recommendation:** Make headers entirely optional. If `_cfheaders.json` doesn't exist:
- Skip the response KVS sync silently (or with a single info-level message).
- Don't error out, don't warn loudly.
- The viewer-response function still works fine with an empty KVS.

Similarly, if there are no redirects beyond directory indexes, that should just work without any `_redirects` file.

---

## Priority 5: Integration with Hugo Workflow (Medium)

### 5.1 Don't try to replace `hugo deploy`

**Problem:** Users already have their own Hugo deployment workflows (CI/CD, `hugo deploy`, manual S3 sync). If Hedgerules tries to do everything, it becomes a heavy dependency.

**Recommendation:** Hedgerules should be complementary:
- `hugo build` — user builds their site (their responsibility).
- `hedgerules deploy` — syncs edge rules to KVS (our responsibility).
- `hugo deploy` — pushes content to S3 (their responsibility).

Hedgerules should NOT invoke `hugo build` or `hugo deploy`. It should clearly document where it fits in the pipeline. A clear mental model: "Hedgerules handles what happens at the CDN edge. Hugo handles what goes in the bucket."

### 5.2 Work as a post-build hook

**Problem:** Users might forget to run `hedgerules deploy` after building.

**Recommendation:** Document how to add Hedgerules as a post-build step:
- **Makefile/Taskfile:** `build: hugo build && hedgerules deploy`
- **CI/CD:** Show a GitHub Actions snippet with both steps.
- **Hugo module:** Maybe in the future, but don't start here — it adds complexity.

Don't auto-detect or auto-run `hugo`. Just make it trivial to add to existing workflows.

---

## Priority 6: Defaults and Conventions (Medium)

### 6.1 Convention-based KVS naming

**Problem:** Users have to manually track KVS names and keep them consistent across CloudFormation, scripts, and config files.

**Recommendation:** Establish a naming convention and document it:
- Request KVS (redirects): `{stack-name}-redirects`
- Response KVS (headers): `{stack-name}-headers`

The CloudFormation template already uses `!Sub "${AWS::StackName}-obverse-redirects"`. We should align with this pattern. For single-site deployments (the common case), just `{stack-name}-redirects`.

### 6.2 Default to current AWS region/profile

**Problem:** Users shouldn't need to specify `--region` if they have AWS configured properly.

**Recommendation:** Use the Go AWS SDK's default credential and region chain. Document this clearly: "Hedgerules uses your default AWS configuration. Set `AWS_REGION` and `AWS_PROFILE` environment variables, or configure `~/.aws/config`." Only provide `--region` and `--profile` flags as overrides.

---

## Priority 7: AWS Resource Creation (Lower)

### 7.1 Don't create CloudFront resources in v1

**Problem:** Automating CloudFront + S3 + KVS resource creation is a huge scope increase. It touches CloudFormation, IAM permissions, DNS, certificates — things that vary wildly per user.

**Recommendation:** For v1, keep resource creation out-of-band. Provide:
1. A well-documented CloudFormation template (already exists in the examples).
2. Clear instructions: "Create these resources first, then use Hedgerules."
3. A `hedgerules check` command that validates the expected resources exist and are accessible.

### 7.2 `hedgerules check` for environment validation

**Problem:** Users will wonder "is my setup correct?" before their first deploy.

**Recommendation:** `hedgerules check` should verify:
- AWS credentials are valid.
- The request KVS exists and is accessible.
- The response KVS exists and is accessible.
- The Hugo output directory exists.
- Print a green checkmark for each, red X for failures with fix instructions.

```
$ hedgerules check
  AWS credentials: OK (account 123456789012)
  Request KVS (my-site-redirects): OK (47 keys)
  Response KVS (my-site-headers): OK (12 keys)
  Hugo output (public/): OK (342 files)
All checks passed. Ready to deploy.
```

---

## Priority 8: Failure Recovery (Lower)

### 8.1 Atomic sync with rollback info

**Problem:** The current Python scripts sync key-by-key. If the process is interrupted mid-sync, the KVS is in a partial state — some keys updated, some stale, some deleted.

**Recommendation:** Use the CloudFront KVS `UpdateKeys` batch API instead of individual `PutKey`/`DeleteKey` calls. This batches all changes into a single API call, making the operation closer to atomic. If the batch fails, nothing changes. If it's too large for a single batch, chunk it and show progress with clear messaging about which chunks succeeded.

### 8.2 Show diff, not just counts

**Problem:** Users want to know exactly what changed, especially after something goes wrong.

**Recommendation:** `hedgerules deploy` should show what's changing, not just counts:
```
Redirects:
  + /new-post -> /new-post/
  ~ /moved-page -> /new-location  (was: /old-location)
  - /deleted-page -> /deleted-page/

Headers:
  + /api/: Cache-Control: no-store
  ~ /: X-Frame-Options: DENY  (was: SAMEORIGIN)
```

Default to a summary. `--verbose` for the full diff. `--quiet` for just counts.

---

## Summary: The Ideal First-Time User Journey

1. User has a Hugo site deploying to S3 + CloudFront.
2. They `go install` hedgerules.
3. They run `hedgerules init --stack-name my-site` in their Hugo project. This creates `hedgerules.toml`.
4. They run `hugo build`.
5. They run `hedgerules deploy --dry-run` to preview.
6. They run `hedgerules deploy` to push.
7. It just works.

Total config: one line (`stack-name`). Total new commands to learn: two (`init` and `deploy`). Total steps beyond their existing workflow: one (`hedgerules deploy` after `hugo build`).
