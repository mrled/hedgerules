---
title: "Comparisons"
weight: 50
---

Hedgerules is designed to work like this:

- Use AWS CloudFront + S3 REST (not S3 website hosting)
- Add redirect and headers templates to your Hugo layouts
- Build your production site with `hugo`
- Deploy the site with `hugo deploy`
- Deploy edge functions with `hedgerules deploy`

## Feature comparison

See the [Feature comparison table]({{< ref "docs/comparisons/table" >}})
for a comparison of Hedgerules with cloud services like
Netlify, AWS Amplify, GCP Firebase, and Azure Static Websites.

## CloudFront + S3: REST vs Websites

AWS offers two ways to serve objects from S3 as a static website,
and it's worth looking at the difference between them.

1.  CloudFront + S3 Website Hosting,
    which requires a public bucket and terminates HTTPS at CloudFront
    (so HTTP from CloudFront to S3).
    It automatically redirects to terminating slashes for requests to directories
    (`/path` -> `/path/`)
    and rewrites directory requests for `index.html`
    (request for `/path/` returns S3 object at `/path/index.html`).

2.  CloudFront + S3 REST,
    which can use a private bucket and talks to S3 over HTTPS.
    It does not automatically handle any kind of directory redirects or rewrites,
    or do any custom header injection by itself.

Both options can use CloudFront Functions to implement redirects, rewrights, and custom headers.

Hedgefront does just that for a CloudFront + S3 REST website.

It's CloudFront + S3 with the ease of Amplify.
