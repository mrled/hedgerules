---
title: "Configuration Reference"
weight: 4
---

Complete reference for all Hedgerules configuration options in `hugo.toml`.

## Output formats

Register the `cfheaders` output format so Hugo generates `_cfheaders.json`:

```toml
[outputs]
  home = ["HTML", "cfheaders"]

[outputFormats]
  [outputFormats.cfheaders]
    baseName = "_cfheaders"
    isPlainText = true
    mediaType = "application/json"
    notAlternative = true
```

To also generate a `_redirects` file from Hugo aliases:

```toml
[outputs]
  home = ["HTML", "cfheaders", "redirects"]

[outputFormats]
  [outputFormats.redirects]
    baseName = "_redirects"
    isPlainText = true
    mediaType = "text/redirects"
    notAlternative = true
```

## Root headers

```toml
[params.CloudFrontNonPagePathHeaders]
  [params.CloudFrontNonPagePathHeaders."/"]
    X-Content-Type-Options = "nosniff"
    X-Frame-Options = "DENY"
```

The root `/` key defines global default headers applied to every response. Only `/` is supported as a key here; per-page headers are defined in frontmatter.

## Per-page headers

In page frontmatter (YAML or TOML):

```yaml
---
CloudFrontHeaders:
  Cache-Control: "public, max-age=86400"
  X-Custom: "value"
---
```

These are added to `_cfheaders.json` keyed by the page's `RelPermalink`. They override root headers with the same name.

## Custom redirects

```toml
[params.redirects]
  "/old" = "/new/"
  "/legacy/url" = "/current/url/"
```

These are written to the `_redirects` file alongside Hugo alias redirects.

## Hugo deploy target

```toml
[deployment]
  [[deployment.targets]]
    name = "mysite"
    URL = "s3://BUCKET?region=REGION"
    cloudFrontDistributionID = "DISTRIBUTION_ID"
```

## File outputs

| File | Description | Consumed by |
|------|-------------|-------------|
| `_cfheaders.json` | Path-to-headers mapping | `hedgerules deploy` (headers KVS) |
| `_redirects` | Source-destination redirect pairs | `hedgerules deploy` (redirects KVS) |

## KVS limits

| Constraint | Limit |
|-----------|-------|
| Max key size | 512 bytes |
| Max key + value size | 1 KB |
| Max total KVS size | 5 MB |
