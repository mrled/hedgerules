#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "boto3[crt]",
# ]
# ///
"""Manage redirects in CloudFront Key Value Store for viewer requests.

This script:
1. Scans directories in the build output to create trailing-slash redirects
2. Reads Hugo's _redirects file for additional redirects
3. Validates size constraints before uploading to CloudFront KVS
"""

import argparse
import sys
from pathlib import Path
from typing import Dict

import boto3  # type: ignore


def get_directory_redirects(output_dir: Path) -> Dict[str, str]:
    """Get redirects for directories (no trailing slash -> with trailing slash)."""
    redirects: Dict[str, str] = {}

    if not output_dir.exists():
        print(f"Error: Output directory does not exist: {output_dir}", file=sys.stderr)
        return redirects

    # Recursively find all directories
    for item in output_dir.rglob("*"):
        if item.is_dir():
            # Get path relative to output_dir
            rel_path = item.relative_to(output_dir)
            # Convert /path/to/dirname -> /path/to/dirname/
            key = f"/{rel_path}"
            value = f"/{rel_path}/"
            redirects[key] = value

    return redirects


def parse_redirects_file(redirects_file: Path) -> Dict[str, str]:
    """Parse Hugo's _redirects file (whitespace-separated)."""
    redirects: Dict[str, str] = {}

    if not redirects_file.exists():
        print(f"Warning: _redirects file not found: {redirects_file}", file=sys.stderr)
        return redirects

    with open(redirects_file, "r") as f:
        for line_num, line in enumerate(f, 1):
            line = line.strip()
            if not line or line.startswith("#"):
                continue

            parts = line.split()
            if len(parts) < 2:
                print(
                    f"Warning: Invalid redirect on line {line_num}: {line}",
                    file=sys.stderr,
                )
                continue

            source = parts[0]
            destination = parts[1]
            redirects[source] = destination

    return redirects


def validate_kvs_data(data: Dict[str, str]) -> bool:
    """Validate CloudFront KVS size constraints.

    Constraints:
    - Maximum key size: 512 bytes
    - Maximum key + value size: 1 KB (1024 bytes)
    - Maximum total data size: 5 MB (5,242,880 bytes)
    """
    total_size = 0
    valid = True

    for key, value in data.items():
        key_bytes = key.encode("utf-8")
        value_bytes = value.encode("utf-8")
        key_size = len(key_bytes)
        value_size = len(value_bytes)
        entry_size = key_size + value_size

        if key_size > 512:
            print(
                f"Error: Key exceeds 512 bytes ({key_size} bytes): {key}",
                file=sys.stderr,
            )
            valid = False

        if entry_size > 1024:
            print(
                f"Error: Key+value exceeds 1 KB ({entry_size} bytes): {key} -> {value}",
                file=sys.stderr,
            )
            valid = False

        total_size += entry_size

    if total_size > 5_242_880:
        print(f"Error: Total data exceeds 5 MB ({total_size} bytes)", file=sys.stderr)
        valid = False

    if valid:
        print(f"✓ Validation passed: {len(data)} entries, {total_size:,} bytes total")

    return valid


def get_kvs_arn(kvs_name: str, region: str | None = None) -> str:
    """Get KVS ARN from name using CloudFront API."""
    cf = (
        boto3.client("cloudfront", region_name=region)
        if region
        else boto3.client("cloudfront")
    )

    # List all KeyValueStores and find the one with matching name
    paginator = cf.get_paginator("list_key_value_stores")
    for page in paginator.paginate():
        for kvs in page.get("KeyValueStoreList", {}).get("Items", []):
            if kvs["Name"] == kvs_name:
                return kvs["ARN"]

    raise ValueError(f"KeyValueStore not found: {kvs_name}")


def list_kvs_keys(kvs_arn: str, region: str | None = None) -> set[str]:
    """List all keys currently in the KVS."""
    client = (
        boto3.client("cloudfront-keyvaluestore", region_name=region)
        if region
        else boto3.client("cloudfront-keyvaluestore")
    )
    keys = set()

    # Get current ETag for the KVS
    client.describe_key_value_store(KvsARN=kvs_arn)

    # List all keys using pagination
    paginator = client.get_paginator("list_keys")
    for page in paginator.paginate(KvsARN=kvs_arn):
        for item in page.get("Items", []):
            keys.add(item["Key"])

    return keys


def sync_to_kvs(kvs_arn: str, data: Dict[str, str], region: str | None = None) -> None:
    """Sync key-value pairs to CloudFront KVS (add/update/delete)."""
    client = (
        boto3.client("cloudfront-keyvaluestore", region_name=region)
        if region
        else boto3.client("cloudfront-keyvaluestore")
    )

    # Get current state
    print("\nFetching current KVS state...")
    response = client.describe_key_value_store(KvsARN=kvs_arn)
    etag = response["ETag"]

    existing_keys = list_kvs_keys(kvs_arn, region)
    new_keys = set(data.keys())

    # Determine operations
    keys_to_delete = existing_keys - new_keys
    keys_to_add = new_keys - existing_keys
    keys_to_update = new_keys & existing_keys

    print("\nSync plan:")
    print(f"  Add: {len(keys_to_add)} keys")
    print(f"  Update: {len(keys_to_update)} keys")
    print(f"  Delete: {len(keys_to_delete)} keys")

    # Delete obsolete keys
    delete_count = 0
    delete_errors = 0
    if keys_to_delete:
        print(f"\nDeleting {len(keys_to_delete)} obsolete keys...")
        for key in sorted(keys_to_delete):
            try:
                response = client.delete_key(KvsARN=kvs_arn, Key=key, IfMatch=etag)
                etag = response["ETag"]
                delete_count += 1
                print(f"✓ Deleted {key}")
            except Exception as e:
                delete_errors += 1
                print(f"✗ Failed to delete {key}: {e}", file=sys.stderr)

    # Add/update keys
    put_count = 0
    put_errors = 0
    if new_keys:
        print(f"\nUploading {len(new_keys)} redirects...")
        for key in sorted(new_keys):
            value = data[key]
            try:
                response = client.put_key(
                    KvsARN=kvs_arn, Key=key, Value=value, IfMatch=etag
                )
                etag = response["ETag"]
                put_count += 1
                action = "Added" if key in keys_to_add else "Updated"
                print(f"✓ {action} {key} -> {value}")
            except Exception as e:
                put_errors += 1
                print(f"✗ Failed to upload {key}: {e}", file=sys.stderr)

    print("\n✓ Sync complete:")
    print(f"  Deleted: {delete_count} succeeded, {delete_errors} failed")
    print(f"  Added/Updated: {put_count} succeeded, {put_errors} failed")


def main():
    parser = argparse.ArgumentParser(
        description="Manage CloudFront KVS redirects for viewer requests"
    )
    parser.add_argument(
        "output_dir",
        type=Path,
        help="Hugo output directory (e.g., public/prod-obverse)",
    )
    parser.add_argument(
        "--kvs-name",
        help="CloudFront KVS name (for future AWS API integration)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Validate and print redirects without uploading",
    )
    parser.add_argument(
        "--region",
        help="AWS region (defaults to AWS CLI configuration)",
    )

    args = parser.parse_args()

    # Step 1: Get directory redirects
    print(f"Scanning directories in: {args.output_dir}")
    dir_redirects = get_directory_redirects(args.output_dir)
    print(f"Found {len(dir_redirects)} directory redirects")

    # Step 2: Parse _redirects file (takes precedence)
    redirects_file = args.output_dir / "_redirects"
    print(f"\nReading redirects file: {redirects_file}")
    file_redirects = parse_redirects_file(redirects_file)
    print(f"Found {len(file_redirects)} file redirects")

    # Step 3: Merge (file redirects take precedence)
    all_redirects = {**dir_redirects, **file_redirects}

    # Report conflicts
    conflicts = set(dir_redirects.keys()) & set(file_redirects.keys())
    if conflicts:
        print(
            f"\nNote: {len(conflicts)} conflicts resolved (file redirects take precedence):"
        )
        for key in sorted(conflicts):
            print(f"  {key}: {dir_redirects[key]} -> {file_redirects[key]}")

    print(f"\nTotal redirects: {len(all_redirects)}")

    # Step 4: Validate
    print("\nValidating KVS constraints...")
    if not validate_kvs_data(all_redirects):
        sys.exit(1)

    # Step 5: Print summary or upload
    if args.dry_run:
        print("\n=== Redirect Summary (Dry Run) ===")
        for key, value in sorted(all_redirects.items()):
            print(f"{key} -> {value}")
    elif not args.kvs_name:
        print("\n=== Redirect Summary ===")
        for key, value in sorted(all_redirects.items()):
            print(f"{key} -> {value}")
        print("\nNote: Use --kvs-name to upload to CloudFront KVS")
    else:
        # Sync to CloudFront KVS
        try:
            kvs_arn = get_kvs_arn(args.kvs_name, args.region)
            print(f"Found KVS: {kvs_arn}")
            sync_to_kvs(kvs_arn, all_redirects, args.region)
        except Exception as e:
            print(f"Error: {e}", file=sys.stderr)
            sys.exit(1)


if __name__ == "__main__":
    main()
