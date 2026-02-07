---
title: "AWS Setup"
weight: 3
---

Hedgerules requires several AWS resources. This guide explains what you need and how to set them up.

## Architecture overview

```
User -> CloudFront Distribution -> S3 Bucket
              |
              +-- Viewer Request Function (redirect + rewrite)
              |       +-- KVS: redirects
              |
              +-- Viewer Response Function (add headers)
                      +-- KVS: headers
```

## Required resources

### S3 bucket

A standard S3 bucket to hold your Hugo site's static files. No static website hosting needed &mdash; CloudFront accesses S3 via the Object Storage API.

```sh
aws s3 mb s3://my-hugo-site-bucket --region us-east-1
```

### CloudFront Distribution

A CloudFront distribution with:
- Origin pointing to your S3 bucket (using Origin Access Control)
- Default cache behavior with the viewer-request and viewer-response functions attached

### CloudFront Key Value Stores

Two KVS instances:
1. **Redirects KVS** &mdash; maps source paths to destination paths, used by the viewer-request function
2. **Headers KVS** &mdash; maps path patterns to header blocks, used by the viewer-response function

### CloudFront Functions

Two CloudFront Functions:
1. **Viewer-request function** &mdash; looks up the URI in the redirects KVS; if found, returns a 301 redirect; otherwise appends `index.html` to trailing-slash URIs
2. **Viewer-response function** &mdash; looks up matching patterns in the headers KVS and adds headers to the response

## Setting up with hedgerules deploy

`hedgerules deploy` handles the CloudFront Functions and KVS for you:

```sh
hedgerules deploy --site public/
```

This command:
1. Creates or updates the two CloudFront Functions
2. Creates or updates the two CloudFront KVS instances
3. Uploads redirect and header entries from your build output

You still need to create the S3 bucket and CloudFront Distribution yourself (or with CloudFormation / Terraform).

## Hugo deploy configuration

Add an S3 deployment target to your `hugo.toml`:

```toml
[deployment]
  [[deployment.targets]]
    name = "mysite"
    URL = "s3://my-hugo-site-bucket?region=us-east-1"
    cloudFrontDistributionID = "E1234567890ABC"

  [[deployment.matchers]]
    pattern = "^.+\\.(js|css|svg|ttf|woff|woff2)$"
    cacheControl = "max-age=31536000, immutable"
    gzip = true

  [[deployment.matchers]]
    pattern = "^.+\\.(png|jpg|jpeg|gif|webp|ico)$"
    cacheControl = "max-age=31536000, immutable"
    gzip = false

  [[deployment.matchers]]
    pattern = "^.+\\.(html|xml|json|txt)$"
    cacheControl = "max-age=3600"
    gzip = true
```

Then deploy:

```sh
hugo deploy --target mysite
```

This uploads files to S3 and invalidates the CloudFront cache.
