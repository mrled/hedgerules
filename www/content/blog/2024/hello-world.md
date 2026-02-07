---
title: "Hello World"
aliases:
  - /hello/
  - /blog/first-post/
CloudFrontHeaders:
  X-Blog-Post: "true"
  Cache-Control: "public, max-age=86400"
---

This is a test blog post. It exercises several Hedgerules features:

- **Hugo aliases**: This page has two aliases (`/hello/` and `/blog/first-post/`), which generate redirect entries in `_redirects`.
- **Custom headers**: This page sets `X-Blog-Post` and `Cache-Control` headers via `CloudFrontHeaders` frontmatter.
- **Nested directory**: Located at `/blog/2024/hello-world/`, this tests deep directory index redirects.

## Test scenarios

| Request | Expected behavior |
|---------|------------------|
| `/blog/2024/hello-world` | 301 redirect to `/blog/2024/hello-world/` |
| `/blog/2024/hello-world/` | 200 with `X-Blog-Post: true` header |
| `/hello/` | 301 redirect to `/blog/2024/hello-world/` |
| `/blog/first-post/` | 301 redirect to `/blog/2024/hello-world/` |
| `/blog/2024` | 301 redirect to `/blog/2024/` |
| `/blog` | 301 redirect to `/blog/` |
