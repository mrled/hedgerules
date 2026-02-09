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

<style>
  .comparison-wrapper {
    overflow-x: auto;
    max-width: 100%;
  }

  table.comparison {
    table-layout: fixed;
    min-width: 800px;
    width: 100%;
    font-size: 85%;

    td,
    td p,
    td ul,
    td li {
      padding: 0;
      margin: 0;
    }
  }

  :target {
    background-color: mark;
  }
</style>

<div class="comparison-wrapper">
<table class="comparison">
  <tr>
    <td></td>
    <td>Hedgerules</td>
    <td>Netlify</td>
    <td>AWS Amplify</td>
    <td>AWS S3 Websites</td>
    <td>Azure Static Web Apps</td>
    <td>Google Firebase Hosting</td>
  </tr>
  <tr>
    <td>Redirects file</td>
    <td><code>_hedge_redirects</code></td>
    <td><a href="https://docs.netlify.com/manage/routing/redirects/overview/#syntax-for-the-_redirects-file">_redirects</a></td>
    <td>None</td>
    <td>None</td>
    <td>
      <code>staticwebapp.config.json</code>'s <code>rewrite</code> and <code>redirect</code>
      <a href="https://learn.microsoft.com/en-us/azure/static-web-apps/configuration#define-routes">rules</a>
    </td>
    <td>
      <code>firebase.json</code>'s rules for
      <a href="https://firebase.google.com/docs/hosting/full-config#redirects">redirects</a> and
      <a href="https://firebase.google.com/docs/hosting/full-config#rewrites">rewrites</a>.
    </td>
  </tr>
  <tr>
    <td>Other redirects methods</td>
    <td>None</td>
    <td><a href="https://docs.netlify.com/build/configure-builds/file-based-configuration/#redirects">redirects section</a> in <code>netlify.toml</code></td>
    <td>
      <ul>
        <li><a href="https://docs.aws.amazon.com/amplify/latest/userguide/redirects.html">AWS console</a></li>
        <li><code>CustomRules</code> in <a href="https://docs.aws.amazon.com/AWSCloudFormation/latest/TemplateReference/aws-resource-amplify-app.html?utm_source=chatgpt.com#aws-resource-amplify-app-properties">CloudFormation template</a></li>
      </ul>
    </td>
    <td>
      <ul>
        <li>Limited <a href="https://docs.aws.amazon.com/AmazonS3/latest/userguide/how-to-page-redirect.html#advanced-conditional-redirects">redirection rules</a></li>
        <li>Limited <a href="https://docs.aws.amazon.com/AmazonS3/latest/userguide/how-to-page-redirect.html#redirect-requests-object-metadata">Website Redirect Location objects</a></li>
        <li>Custom CloudFront function<sup><a href="#cloudfront-functions">#</a></sup></li>
      </ul>
    </td>
    <td></td>
    <td></td>
  <tr>
    <td>Headers file</td>
    <td><code>_hedge_headers.json</code></td>
    <td><a href="https://docs.netlify.com/manage/routing/headers/">_headers</a></td>
    <td>None</td>
    <td>None</td>
    <td>
      <code>staticwebapp.conig.json</code>'s
      <a href="https://learn.microsoft.com/en-us/azure/static-web-apps/configuration#global-headers">global headers</a>
      and <a href="https://learn.microsoft.com/en-us/azure/static-web-apps/configuration#route-headers">route headers</a>
    </td>
    <td></td>
  </tr>
  <tr>
    <td>Other headers methods</td>
    <td>None</td>
    <td><a href="https://docs.netlify.com/build/configure-builds/file-based-configuration/#headers">headers section</a> in <code>netlify.toml</code></td>
    <td><a href="https://docs.aws.amazon.com/amplify/latest/userguide/custom-headers.html">customHttp.yml</a>
    <td>
      <ul>
        <li>Limited headers via <a href="https://docs.aws.amazon.com/AmazonS3/latest/userguide/UsingMetadata.html">object metadata</a></li>
        <li>Limited <a href="https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/understanding-response-headers-policies.html">CloudFront Response Headers Policy</a></li>
        <li>Custom CloudFront function<sup><a href="#cloudfront-functions">#</a></sup></li>
      </ul>
    </td>
    <td></td>
    <td></td>
  <tr>
    <td><a href="/docs/headers/request-tokens">Request token</a> in headers</td>
    <td><code>{/path}</code></td>
    <td>None</td>
    <td>None</td>
    <td>None</td>
    <td></td>
    <td></td>
  </tr>
  <tr>
    <td>Deployment methods</td>
    <td>
      <code>hugo deploy && hedgerules deploy</code>
    </td>
    <td>
      <ul>
        <code>git push</code>
        or
        <code>netlify deploy</code>
      </ul>
    </td>
    <td>
      <code>git push</code>,
      then change redirects out of band
    </td>
    <td>
      <code>hugo deploy</code>,
      then change redirects/headers out of band
    </td>
    <td></td>
    <td></td>
  </tr>
  <tr>
    <td>Access logs</td>
    <td>
      <a href="https://docs.aws.amazon.com/athena/">AWS Athena</a>,
      free<sup><a href="#athena-pricing">#</a></sup>
    </td>
    <td>
      <ul>
        <li>Raw access logs via <a href="https://docs.netlify.com/manage/monitoring/log-drains/">Log Drains</a> (enterprise-only "contact us" pricing)</li>
        <li><a href="https://docs.netlify.com/manage/monitoring/web-analytics/how-web-analytics-works/">Web Analytics</a> shows data based on raw logs like unique visitors, top locations, etc ($9/mo/site)</li>
      </ul>
    </td>
    <td>
      <a href="https://docs.aws.amazon.com/athena/">AWS Athena</a>,
      free<sup><a href="#athena-pricing">#</a></sup>
    </td>
    <td>
      <a href="https://docs.aws.amazon.com/athena/">AWS Athena</a>,
      free<sup><a href="#athena-pricing">#</a></sup>
    </td>
    <td></td>
    <td></td>
  </tr>
</table>
</div>

### CloudFront Functions

S3 Websites can use CloudFront functions to implement custom logic
for directory redirects, index rewrites, and custom headers.

### Athena pricing

The Athena feature is free,
but logs and Athena query results both incur storage costs on S3,
and querying logs via Athena incurs execution costs.

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
