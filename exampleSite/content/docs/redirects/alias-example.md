---
title: "Alias Example"
aliases:
  - /old-redirect-docs/
  - /docs/legacy-redirects/
---

This page has Hugo aliases that test redirect generation.

## Expected redirects

The following redirects should be generated in `_redirects`:

| Source | Destination |
|--------|-------------|
| `/old-redirect-docs/` | `/docs/redirects/alias-example/` |
| `/docs/legacy-redirects/` | `/docs/redirects/alias-example/` |

These are picked up by `hedgerules deploy` and stored in the redirects KVS.
