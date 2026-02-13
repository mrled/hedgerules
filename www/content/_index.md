---
title: "Hedgerules"
type: docs
HedgerulesHeaders:
  X-Hedgerules-Hello: "Nice to meet you"
  X-Hedgerules-Local-Development-Path: "http://localhost:1313{/path}"
---

Hugo Edge Rules for AWS CloudFront.

Hedgerules makes `hugo deploy` work seamlessly with CloudFront and S3.
It generates the CloudFront Functions and Key Value Store entries needed to handle
redirects, directory indices, and custom headers --- features you get for free
with Netlify, but have to build yourself on AWS.

## Features

- **Directory index redirects** --- `/path` redirects to `/path/`
- **Index file rewriting** --- `/path/` serves `/path/index.html` from S3
- **Hugo alias redirects** --- redirects from Hugo's `aliases` frontmatter
- **Custom redirects** --- user-defined `_hedge_redirects.txt` file
- **Custom response headers** --- global defaults and per-page overrides via `_hedge_headers.json`

## Quick start

1. [Getting Started guide]({{< relref "/docs/guides/getting-started" >}})
2. [Full documentation]({{< relref "/docs" >}})
