---
title: "AI Agents"
weight: 50
---

# Working on Hedgerules with AI agents

This page documents how AI coding agents (Claude Code, Cursor, Copilot, etc.) should approach working on the Hedgerules codebase.

## Project structure

Hedgerules has three main components:

- **`hedgerules/`** -- the Go CLI application. See [Architecture]({{< relref "architecture" >}}) for package layout and data flow.
- **`hugo-theme-hedgerules/`** -- a Hugo theme that provides output format templates for `_hedge_headers.json` and `_hedge_redirects.txt`.
- **`www/`** -- this documentation site, built with Hugo and the [hugo-book](https://github.com/alex-shpak/hugo-book) theme.

Infrastructure lives in `infra/` and an example deployment lives in `examples/micahrlweb/`.

## Key design decisions

Before making changes, read [Design Decisions]({{< relref "decisions" >}}). The project follows a minimalist philosophy:

- No features without clear justification for maintenance cost.
- stdlib over third-party libraries where reasonable (`flag` over cobra, no logging framework).
- Test pure data transformations, not AWS plumbing.

## Documentation site conventions

The documentation under `www/content/docs/` uses Hugo with YAML frontmatter. Key conventions:

- Every section has an `_index.md` with a `title` and `weight` for ordering.
- Per-page custom headers use `HedgerulesHeaders` in frontmatter (see [Custom Headers]({{< relref "/docs/headers" >}})).
- Redirect aliases use Hugo's built-in `aliases` frontmatter (see [Redirects]({{< relref "/docs/redirects" >}})).
- The [Configuration Reference]({{< relref "/docs/configuration" >}}) documents all `hugo.toml` options.
- The [Getting Started]({{< relref "/docs/guides/getting-started" >}}) guide is the main onboarding path.

## Go CLI conventions

- Go 1.21+, using `go:embed` for CloudFront Function JS files.
- AWS SDK v2 for CloudFront and CloudFront KeyValueStore operations.
- TOML config via `BurntSushi/toml` (Hugo users already know TOML).
- Validate all data before making any AWS API calls. Collect all validation errors and report them together.
- No retries on AWS errors -- re-running `hedgerules deploy` is safe because KVS sync is convergent.

## Running the site locally

```sh
cd www
hugo server
```

The site builds to `www/public/development/` in development mode and `www/public/production/` in production mode. See `www/config/` for environment-specific Hugo configuration.

## Building the CLI

```sh
cd hedgerules
go build ./cmd/hedgerules
```
