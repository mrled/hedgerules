---
title: "Per-Page Header Example"
HedgerulesHeaders:
  X-Custom-Page: "per-page-example"
---

This page tests per-page CloudFront headers.

## Expected headers

When this page is served, the response should include:

| Header | Value | Source |
|--------|-------|--------|
| `X-Custom-Page` | `per-page-example` | This page's frontmatter |
| `X-Hedgerules-Hello` | `Nice to meet you` | Root `/` defaults |
| `X-Hedgerules-Local-Development-Path` | `http://localhost:1313/docs/headers/per-page-example/` | Root `/` defaults (request path token) |
| `X-Hedgerules-Hello-Subpath` | `Hello from this specific subpath (and its children)` | Parent `/docs/headers/` |

The viewer-response function checks multiple KVS patterns: exact path, parent directories, and root `/`. More-specific headers take priority.
