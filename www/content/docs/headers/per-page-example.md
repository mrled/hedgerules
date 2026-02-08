---
title: "Per-Page Header Example"
HedgerulesHeaders:
  X-Custom-Page: "per-page-example"
  Cache-Control: "public, max-age=86400"
---

This page tests per-page CloudFront headers.

## Expected headers

When this page is served, the response should include:

| Header | Value | Source |
|--------|-------|--------|
| `X-Custom-Page` | `per-page-example` | This page's frontmatter |
| `Cache-Control` | `public, max-age=86400` | This page's frontmatter |
| `X-Content-Type-Options` | `nosniff` | Root `/` defaults |
| `X-Frame-Options` | `DENY` | Root `/` defaults |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Root `/` defaults |

The viewer-response function performs two KVS lookups: one for the exact page path, one for root `/`. Headers from the exact path take priority over root defaults.
