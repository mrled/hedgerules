---
title: "Pricing Estimation"
weight: 50
---

# Pricing Estimation

## S3

TBD

## CloudFront Distribution

TBD

## CloudFront KVS

Hedgerules uses CloudFront KeyValueStore (KVS) for redirects and custom response headers.
KVS charges for API operations: reads happen on every request at the edge,
and writes happen when you run `hedgerules deploy`.

<table>
  <thead>
    <tr>
      <th>Operation</th>
      <th>Rate</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Reads from CloudFront Functions at request time</td>
      <td>$0.03 per 1 million</td>
    </tr>
    <tr>
      <td>Writes during deployment</td>
      <td>$1.00 per 1,000</td>
    </tr>
  </tbody>
</table>

### Writes during deployment

Each `hedgerules deploy` writes to two KVS stores.

**Redirects KVS:**

<table>
  <thead>
    <tr>
      <th>Source</th>
      <th>Writes</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Directory rewrites</td>
      <td>1 per Hugo page (e.g. <code>/blog</code> &rarr; <code>/blog/</code>)</td>
    </tr>
    <tr>
      <td>Redirects</td>
      <td>1 per Hugo alias or custom redirect</td>
    </tr>
  </tbody>
</table>

**Headers KVS:**

<table>
  <thead>
    <tr>
      <th>Source</th>
      <th>Writes</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Header rules</td>
      <td>1 per user-defined pattern (e.g. <code>/</code>, <code>/blog/</code>, <code>*.html</code>)</td>
    </tr>
  </tbody>
</table>

KVS writes are performed in batches of 50 per store.
So a site with 50 pages will create 50 directory rewrite entries,
but will consume just a single API call to apply them.

Each [header rule](/docs/headers/) is a single KVS entry:
the key is the path pattern and the value contains
all headers for that pattern packed together.
(A single path pattern that applies multiple headers is still just one KVS entry.)

Hedgerules uses a convergent sync: it diffs the desired state against the existing KVS contents
and only writes entries that have changed.
Only the first deploy and deploys with actual changes incur write costs.

#### Example

A site with:

- 80 directories (including `/`, `/blog/`, `/blog/2024/`, `/docs/`, etc.)
- 15 redirect aliases
- 4 header rules (`/`, `*.html`, `/blog/`, `/docs/`)

Would generate at most **95 redirects KVS entries** and **4 headers KVS entries** per deploy.

With 50 operations per API call per store,
this is three API calls:
- two against the redirects KVS to handle 80+15 entries
- one against the headers KVS to handle 4 entries

### Reads during deployment

Before writing, `hedgerules deploy` reads the full existing state of each KVS store to compute a diff.
Per store this requires:

- 1 `DescribeKeyValueStore` call (fetches the ETag needed for the subsequent write)
- &lceil;N&divide;50&rceil; `ListKeys` calls, where N is the number of existing entries (page size: 50)

#### Example

For the same site (95 redirect entries, 4 header entries, on a typical non-first deploy):

- Redirects KVS: 1 describe + 2 list calls (&lceil;95&divide;50&rceil;&nbsp;=&nbsp;2)
- Headers KVS: 1 describe + 1 list call (&lceil;4&divide;50&rceil;&nbsp;=&nbsp;1)
- Total: **6 read calls** per deploy

At $0.03/1M reads this cost is negligible.

### Reads per request

Every request triggers two CloudFront Functions, each reading from its KVS.

#### Viewer-request (redirects KVS)

One read per request: the exact URI is looked up in the redirects KVS.
If a redirect is found, a 301 is returned immediately.
If not, the request continues to the origin (and viewer-response runs on the way back).

#### Viewer-response (headers KVS)

Multiple reads per request, checked in order from least to most specific:

1. Root `/` (always)
2. Each parent directory, with trailing slash (e.g. `/blog/`, `/blog/post/`)
3. Extension wildcard, if the file has an extension (e.g. `*.html`, `*.xml`)
4. Exact file path (e.g. `/blog/post/hello/index.html`)

For a request to `/blog/post/hello/`
(which viewer-request rewrites to `/blog/post/hello/index.html` before the origin fetch),
viewer-response checks these patterns in order:

```
/
/blog/
/blog/post/
/blog/post/hello/
*.html
/blog/post/hello/index.html
```

That is **6 reads** from the headers KVS.

The general formula for a file at path depth _D_ (number of `/`-separated segments, including the filename) is:

> **D + 2 reads** (root + D&minus;1 parent directories + extension + exact path)

For a path like `/blog/post/hello/index.html`, D&nbsp;=&nbsp;4, so 4&nbsp;+&nbsp;2&nbsp;=&nbsp;6 reads.
For a top-level file like `/feed.xml`, D&nbsp;=&nbsp;1, so 1&nbsp;+&nbsp;2&nbsp;=&nbsp;3 reads.
For the root `/` itself, only the root pattern is checked: 1 read.

If the file has no extension (uncommon in Hugo output), the extension wildcard step is skipped,
reducing the count by 1.

#### Total reads per request

Adding the viewer-request redirect lookup: **D&nbsp;+&nbsp;3 reads per request** for most pages,
where D is the path depth.

#### Example

A site receiving 500,000 requests per month,
where the average page is 3 levels deep (D&nbsp;=&nbsp;4, so 4+3=7 reads per request):

- 500,000 &times; 7 = 3,500,000 reads/month
- At $0.03/1M: roughly **$0.11/month** or **$1.30/year**.
