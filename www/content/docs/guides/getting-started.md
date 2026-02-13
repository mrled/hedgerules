---
title: "Getting Started"
weight: 1
aliases:
  - /guides/getting-started/
  - /test/redirect-getting-started
---

This guide walks you through deploying a Hugo site to AWS CloudFront using Hedgerules.

## Prerequisites

- [Hugo](https://gohugo.io/) installed
- An AWS account with permissions for S3, CloudFront, and CloudFront KeyValueStore
- AWS CLI configured

## 1. Install Hedgerules

```sh
go install github.com/yourorg/hedgerules@latest
```

## 2. Add the _hedge_headers.json template

Copy the `index.hedgeheaders.json` layout from the Hedgerules example site into your Hugo site's `layouts/` directory. This template generates a `_hedge_headers.json` file in your build output that maps URL paths to HTTP response headers.

You also need to register the output format in your `hugo.toml`:

```toml
[outputs]
  home = ["HTML", "hedgeheaders", "hedgeredirects"]
```

The output format definitions are provided by the theme. If you're not using the theme, see the [Configuration Reference](/docs/configuration/) for the full output format config.

## 3. Configure headers in hugo.toml

Add global default headers under the root `/` key:

```toml
[params.HedgerulesPathHeaders]
  [params.HedgerulesPathHeaders."/"]
    X-Content-Type-Options = "nosniff"
    X-Frame-Options = "DENY"
```

These apply to every response. Override them for specific pages using frontmatter:

```yaml
---
title: "My Page"
HedgerulesHeaders:
  Cache-Control: "max-age=3600"
---
```

## 4. Build your Hugo site

```sh
hugo
```

This produces `public/` with `_hedge_headers.json` and `_hedge_redirects.txt` alongside your HTML.

## 5. Deploy edge rules

```sh
hedgerules deploy --site public/
```

This creates (or updates) the CloudFront Functions and KVS entries for redirects and headers.

## 6. Deploy site content

```sh
hugo deploy --target mysite
```

This uploads your site to S3 and invalidates the CloudFront cache.

## What Hedgerules does

When you run `hedgerules deploy`, it:

1. **Scans directories** in your build output to generate index redirects (`/path` -> `/path/`)
2. **Reads `_hedge_redirects.txt`** for Hugo alias redirects and custom redirects
3. **Reads `_hedge_headers.json`** for custom response headers
4. **Uploads redirect entries** to a CloudFront KVS (used by the viewer-request function)
5. **Uploads header entries** to a CloudFront KVS (used by the viewer-response function)

The CloudFront Functions handle the actual request/response processing at the edge.
