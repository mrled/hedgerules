---
title: "Configuration Reference"
weight: 60
---

Complete reference for all Hedgerules configuration options in `hugo.toml`.

## Output formats

Enable the Hedgerules output formats on the home page:

```toml
[outputs]
  home = ["HTML", "hedgeheaders", "hedgeredirects"]
```

If you're using the `hugo-theme-hedgerules` theme, the format definitions are provided automatically. If not, add them manually:

```toml
[outputFormats]
  [outputFormats.hedgeheaders]
    baseName = "_hedge_headers"
    isPlainText = true
    mediaType = "application/json"
    notAlternative = true
  [outputFormats.hedgeredirects]
    baseName = "_hedge_redirects"
    isPlainText = true
    mediaType = "text/plain"
    notAlternative = true
```

## Root headers

```toml
[params.HedgerulesPathHeaders]
  [params.HedgerulesPathHeaders."/"]
    X-Content-Type-Options = "nosniff"
    X-Frame-Options = "DENY"
```

The root `/` key defines global default headers applied to every response. Only `/` is supported as a key here; per-page headers are defined in frontmatter.

## Per-page headers

In page frontmatter (YAML or TOML):

```yaml
---
HedgerulesHeaders:
  Cache-Control: "public, max-age=86400"
  X-Custom: "value"
---
```

These are added to `_hedge_headers.json` keyed by the page's `RelPermalink`. They override root headers with the same name.

## Custom redirects

```toml
[params.redirects]
  "/old" = "/new/"
  "/legacy/url" = "/current/url/"
```

These are written to `_hedge_redirects.txt` alongside Hugo alias redirects.

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
| `_hedge_headers.json` | Path-to-headers mapping | `hedgerules deploy` (headers KVS) |
| `_hedge_redirects.txt` | Source-destination redirect pairs | `hedgerules deploy` (redirects KVS) |

## KVS limits

| Constraint | Limit |
|-----------|-------|
| Max key size | 512 bytes |
| Max key + value size | 1 KB |
| Max total KVS size | 5 MB |
