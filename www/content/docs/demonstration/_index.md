---
title: "Demonstration"
weight: 15
---

Hedgerules is used to deploy this site.
Here is a demonstration of how it works.

See how this site is deployed by looking at the [hedgerules.micahrl.com example]({{< relref "/docs/examples/hedgerules.micahrl.com" >}}).

## Headers on all pages

The root page has headers for all pages on the site:

```sh
curl -I https://hedgerules.micahrl.com/
# x-hedgerules-hello: Nice to meet you

curl -I https://hedgerules.micahrl.com/docs/deployment/
# x-hedgerules-hello: Nice to meet you
```

## Headers on all pages with a request path token

[Request path token]({{< relref "/docs/headers/request-path-tokens" >}})
contain the request path in them.
Any page on the site will show what its default local Hugo development mode URL would be:

```sh
curl -I https://hedgerules.micahrl.com/
# x-hedgerules-local-development-path: http://localhost:1313/

curl -I https://hedgerules.micahrl.com/docs/deployment/
# x-hedgerules-local-development-path: http://localhost:1313/docs/deployment/

# ... etc
```

## Subpath headers

Pages under `/docs/headers/` inherit headers from the section's `_index.md`:

```sh
curl -I https://hedgerules.micahrl.com/docs/headers/
# x-hedgerules-hello-subpath: Hello from this specific subpath (and its children)
```

## Path headers

Any path can have custom headers set via `HedgerulesPathHeaders` in `hugo.toml`:

```sh
curl -I https://hedgerules.micahrl.com/favicon.svg
# x-hedgerules-icon-about: The icon is made of the characters 'h≡', in JetBrains Mono, size 56, color '#14532D'
```

## Wildcard path headers

You can set headers for all files of a specific type.
For instance, we set a content type and options on all XML files:

```sh
curl -I https://hedgerules.micahrl.com/sitemap.xml
# content-type: application/xml; charset=utf-8
# x-content-type-options: nosniff
```

## Redirects

Hugo alias redirects are handled by the CloudFront viewer-request function.
Here's a real redirect we have set up for you to see:

```sh
curl -LI https://hedgerules.micahrl.com/test/redirect-getting-started
# HTTP/2 301 (redirect to /test/redirect-getting-started/ with a trailing slash)
# HTTP/2 301 (redirect to /docs/guides/getting-started/ the final destination)
# HTTP/2 200
```

## Index rewrites

Index rewrites are automatically generated from the build output:

```sh
curl -LI https://hedgerules.micahrl.com/docs/deployment/
# HTTP/2 200 (returns the content from /docs/deployment/index.html)
```
