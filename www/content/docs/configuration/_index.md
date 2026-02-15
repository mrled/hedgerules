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

## Path headers

```toml
[params.HedgerulesPathHeaders]
  [params.HedgerulesPathHeaders."/favicon.svg"]
    X-Hedgerules-Icon-About = "The icon is made of the characters 'h≡', in JetBrains Mono, size 56, color '#14532D'"
  [params.HedgerulesPathHeaders."/favicon.png"]
    X-Hedgerules-Icon-About = "The icon is made of the characters 'h≡', in JetBrains Mono, size 56, color '#14532D'"
  [params.HedgerulesPathHeaders."*.xml"]
    Content-Type = "application/xml; charset=utf-8"
    X-Content-Type-Options = "nosniff"
```

You can set any path here,
including specific files,
directories (which would include all children),
and wildcards.

## Per-page headers

In page frontmatter (YAML or TOML):

```yaml
---
HedgerulesHeaders:
  Cache-Control: "public, max-age=86400"
  X-Custom: "value"
---
```

These are added to `_hedge_headers.json` keyed by the page's `RelPermalink`.
They override root headers with the same name.
They apply to all child paths too.

## Custom redirects

```toml
[params.HedgerulesRedirects]
  "/old" = "/new/"
  "/legacy/url" = "/current/url/"
```

Site-wide redirects written to `_hedge_redirects.txt` before Hugo alias and per-page redirects.

## Per-page path redirects

In page frontmatter:

```yaml
---
HedgerulesPathRedirects:
  - from: /shortlink
    to: child-file.pdf
  - from: /other
    to: /absolute/path/
---
```

Each entry creates a redirect from `from` to a destination derived from `to`.
If `to` starts with `/`, it is used as an absolute path.
Otherwise, it is prefixed with the page's `RelPermalink`.
This is useful for creating redirects to files without frontmatter (images, PDFs, etc.).

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
