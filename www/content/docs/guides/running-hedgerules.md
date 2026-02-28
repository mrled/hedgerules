---
title: "Running hedgerules"
weight: 2
---

`hedgerules deploy` reads your Hugo build output and syncs redirect and header data to CloudFront KVS, then deploys the CloudFront Functions that serve it.

## Config file

By default, `hedgerules` reads `hedgerules.toml` from the current directory. All fields are optional in the file; any omitted field can be supplied via CLI flag instead.

```toml
# hedgerules.toml

output-dir = "public/"
region = "us-east-1"
redirects-kvs-name = "mysite-redirects"
headers-kvs-name = "mysite-headers"
viewer-request-name = "mysite-viewer-request"
viewer-response-name = "mysite-viewer-response"
# debug-headers = false
# max-retries = 10
```

| Key | Description |
|---|---|
| `output-dir` | Hugo build output directory |
| `region` | AWS region |
| `redirects-kvs-name` | CloudFront KVS name for redirect data |
| `headers-kvs-name` | CloudFront KVS name for header data |
| `viewer-request-name` | CloudFront Function name for viewer-request |
| `viewer-response-name` | CloudFront Function name for viewer-response |
| `debug-headers` | Inject debug headers into viewer-response (default `false`) |
| `max-retries` | Max retries on AWS throttling errors (default `10`, `0` disables retries) |

## Command line flags

CLI flags override config file values. All string flags support `@FILE` syntax (see below).

```
hedgerules deploy [flags]
```

| Flag | Description |
|---|---|
| `--output-dir` | Hugo build output directory |
| `--region` | AWS region override |
| `--redirects-kvs-name` | CloudFront KVS name for redirect data |
| `--headers-kvs-name` | CloudFront KVS name for header data |
| `--request-function-name` | CloudFront Function name for viewer-request |
| `--response-function-name` | CloudFront Function name for viewer-response |
| `--debug-headers` | Inject debug headers into viewer-response |
| `--max-retries` | Max retries on AWS throttling errors (default `10`, `0` disables retries) |
| `--dry-run` | Parse and validate only; print plan without mutating AWS |
| `--config` | Path to config file (default: `hedgerules.toml`) |

Required fields (`output-dir`, `redirects-kvs-name`, `headers-kvs-name`, `request-function-name`, `response-function-name`) must be set by either the config file or CLI flags. With `--dry-run`, only `output-dir` is required.

## Resolution order

Values are resolved in this order, with later sources taking precedence:

1. Config file (`hedgerules.toml`, or the path given by `--config`)
2. CLI flags

## @FILE syntax

Any string flag (and the corresponding config file value is not affected — this applies to CLI flags only) accepts a `@FILE` value. If a flag value starts with `@`, the remainder is treated as a file path and the flag's value is read from that file, with leading and trailing whitespace stripped.

```sh
hedgerules deploy --region @/run/secrets/aws-region
```

This is useful for passing secrets from files or secret stores without exposing them on the command line:

```sh
hedgerules deploy \
  --redirects-kvs-name @.kvs-redirects-name \
  --headers-kvs-name @.kvs-headers-name \
  --request-function-name @.function-request-name \
  --response-function-name @.function-response-name
```

It's also useful for orchestration by tools like `make`,
which might retrieve values from CloudFormation stack deploymenst
and use `jq` to save each value as a separate file,
and can pass those values easily to Hedgerules.

If the file cannot be read, `hedgerules` exits with an error.
