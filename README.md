# Hedgerules

Hedgerules (Hugo Edge Rules) makes `hugo deploy` work seamlessly with CloudFront and S3.

It was created to provide directory redirects, index rewrites, and custom headers for AWS-hosted Hugo sites,
to enable migration from Netlify which implements those features natively.

When using CloudFront with S3 REST (private bucket, HTTPS to S3),
you have to implement directory index redirects, index file rewriting,
custom headers, and redirects yourself.
Hedgerules implements these features using CloudFront Functions
and CloudFront Key Value Store (KVS).

See <https://hedgerules.micahrl.com> for complete documentation and examples.
That site is deployed with Hedgerules!

Deployment workflow:

1. Create S3 bucket, CloudFront Distribution, and KVS resources out of band (e.g. CloudFormation).
2. Run `hugo` to build the site, generating `_hedge_redirects.txt` and `_hedge_headers.json`.
3. Run `hedgerules deploy` to create/update CloudFront Functions and sync KVS entries.
4. Run `hugo deploy` to publish the site to S3 and invalidate the CloudFront cache.

See the [deployment docs](https://hedgerules.micahrl.com/docs/deployment/) for the full 5-stage setup.

## Contributing

Contributions welcome.
I'm especially interested in:

- Supporting more clouds
- Some form of HTTP basic authentication
- Other features that would be useful and free (or basically free) for personal-scale websites
