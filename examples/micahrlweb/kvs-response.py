#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "boto3[crt]",
# ]
# ///
"""Manage custom headers in CloudFront Key Value Store for viewer responses.

This script:
1. Reads Hugo's _cfheaders.json file for custom headers
2. Validates size constraints before uploading to CloudFront KVS
"""

import argparse
import json
import sys
from pathlib import Path
from typing import Dict

import boto3  # type: ignore


def parse_headers_file(headers_file: Path) -> Dict[str, str]:
    """Parse _cfheaders.json file.

    Format:
        {
          "/path/to/somewhere": {
            "Header-Name": "Some header value",
            "Another-Header": "Another value"
          },
          "/another/path": {
            "X-Custom": "value"
          }
        }

    Returns dict mapping paths to their header blocks (newline-separated string).
    """
    headers: Dict[str, str] = {}

    if not headers_file.exists():
        print(f"Error: Headers file not found: {headers_file}", file=sys.stderr)
        return headers

    with open(headers_file, "r") as f:
        data = json.load(f)

    # Convert JSON format to the expected string format
    for path, header_dict in data.items():
        # Each header should be formatted as "Name: Value"
        header_lines = [f"{name}: {value}" for name, value in header_dict.items()]
        headers[path] = "\n".join(header_lines)

    return headers


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
                f"Error: Key+value exceeds 1 KB ({entry_size} bytes): {key}",
                file=sys.stderr,
            )
            print(f"  Key: {key_size} bytes", file=sys.stderr)
            print(f"  Value: {value_size} bytes", file=sys.stderr)
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
        print(f"\nUploading {len(new_keys)} header entries...")
        for key in sorted(new_keys):
            value = data[key]
            try:
                response = client.put_key(
                    KvsARN=kvs_arn, Key=key, Value=value, IfMatch=etag
                )
                etag = response["ETag"]
                put_count += 1
                action = "Added" if key in keys_to_add else "Updated"
                print(f"✓ {action} {key}")
            except Exception as e:
                put_errors += 1
                print(f"✗ Failed to upload {key}: {e}", file=sys.stderr)

    print("\n✓ Sync complete:")
    print(f"  Deleted: {delete_count} succeeded, {delete_errors} failed")
    print(f"  Added/Updated: {put_count} succeeded, {put_errors} failed")


def main():
    parser = argparse.ArgumentParser(
        description="Manage CloudFront KVS custom headers for viewer responses"
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
        help="Validate and print headers without uploading",
    )
    parser.add_argument(
        "--region",
        help="AWS region (defaults to AWS CLI configuration)",
    )

    args = parser.parse_args()

    # Step 1: Parse _cfheaders.json file
    headers_file = args.output_dir / "_cfheaders.json"
    print(f"Reading headers file: {headers_file}")
    headers = parse_headers_file(headers_file)

    if not headers:
        print("No headers found")
        sys.exit(1)

    print(f"Found {len(headers)} header entries")

    # Step 2: Validate
    print("\nValidating KVS constraints...")
    if not validate_kvs_data(headers):
        sys.exit(1)

    # Step 3: Print summary or upload
    if args.dry_run:
        print("\n=== Headers Summary (Dry Run) ===")
        for path, header_block in sorted(headers.items()):
            print(f"\n{path}")
            print(header_block)
            print("---")
    elif not args.kvs_name:
        print("\n=== Headers Summary ===")
        for path, header_block in sorted(headers.items()):
            print(f"\n{path}")
            print(header_block)
            print("---")
        print("\nNote: Use --kvs-name to upload to CloudFront KVS")
    else:
        # Sync to CloudFront KVS
        try:
            kvs_arn = get_kvs_arn(args.kvs_name, args.region)
            print(f"Found KVS: {kvs_arn}")
            sync_to_kvs(kvs_arn, headers, args.region)
        except Exception as e:
            print(f"Error: {e}", file=sys.stderr)
            sys.exit(1)


if __name__ == "__main__":
    main()
