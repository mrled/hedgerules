---
title: "Hugo Theme"
weight: 55
---

The `hugo-theme-hedgerules` Hugo theme provides output format layouts, data partials, and table partials for working with Hedgerules headers and redirects.

## Output format layouts

These templates generate the files that `hedgerules deploy` reads.

### `index.hedgeheaders.json`

Generates `_hedge_headers.json`, a JSON file mapping paths to HTTP header objects. Enable it via `[outputs]` in `hugo.toml`:

```toml
[outputs]
  home = ["HTML", "hedgeheaders", "hedgeredirects"]
```

### `index.hedgeredirects.txt`

Generates `_hedge_redirects.txt`, a text file with one `source destination` pair per line. Enable it alongside `hedgeheaders` as shown above.

## Data partials

These partials return data and can be called from your own templates or shortcodes.

### `hedgerules/redirects.html`

Returns a map of all redirects (source path to destination path). Sources are collected in this order:

1. Site-wide `HedgerulesRedirects` from `hugo.toml`
2. Hugo `aliases` from page frontmatter
3. `HedgerulesPathRedirects` from page frontmatter

The result is cached on `hugo.Store` so it is only computed once per build.

```go-html-template
{{ $redirects := partial "hedgerules/redirects.html" . }}
{{ range $source, $dest := $redirects }}
  {{ $source }} -> {{ $dest }}
{{ end }}
```

### `hedgerules/headers.html`

Returns a map of all headers (path to header dict). Sources are merged in this order (later wins):

1. `HedgerulesPathHeaders` from `hugo.toml`
2. `HedgerulesHeaders` from page frontmatter

The result is cached on `hugo.Store` so it is only computed once per build.

```go-html-template
{{ $headers := partial "hedgerules/headers.html" . }}
{{ range $path, $headerDict := $headers }}
  {{ $path }}:
  {{ range $name, $value := $headerDict }}
    {{ $name }}: {{ $value }}
  {{ end }}
{{ end }}
```

## Table partials and shortcodes

These partials render HTML tables and are useful for documentation or debugging pages.

### `hedgerules/redirectsTable.html`

Renders an HTML table with Source and Destination columns for all redirects.

```go-html-template
{{ partial "hedgerules/redirectsTable.html" . }}
```

This is also available as a shortcode.

```markdown
{{</* hedgerules/redirectsTable */>}}
```

On this site, it renders like this:

{{< hedgerules/redirectsTable >}}

### `hedgerules/headersTable.html`

Renders an HTML table with Path, Header, and Value columns for all headers.

```go-html-template
{{ partial "hedgerules/headersTable.html" . }}
```

This is also available as a shortcode.

```markdown
{{</* hedgerules/headersTable */>}}
```

On this site, it renders like this:

{{< hedgerules/headersTable >}}
