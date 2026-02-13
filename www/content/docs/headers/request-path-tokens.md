---
title: "Request Path Tokens"
aliases:
  - /docs/headers/request-tokens/
---

Header values can include **request path tokens** that are replaced at request time by the CloudFront viewer-response function.

## Available tokens

| Token | Replaced with |
|-------|---------------|
| `{/path}` | The request path (e.g., `/docs/headers/request-path-tokens/`) |

## Example: Onion-Location

The [`Onion-Location`](https://community.torproject.org/onion-services/ecosystem/apps/web/onion-location/) header tells Tor Browser that your site is available as an onion service. The header value must include the full `.onion` URL for the current page.

With request path tokens, you can set this once on root `/` and have it work for every page:

```toml
[params.HedgerulesPathHeaders]
  [params.HedgerulesPathHeaders."/"]
    Onion-Location = "http://example2255goo6ofqontxj.onion{/path}"
```

A request to `/docs/headers/` produces the response header:

```
Onion-Location: http://example2255goo6ofqontxj.onion/docs/headers/
```

Without request path tokens, you would need to set `Onion-Location` individually in every page's frontmatter.
