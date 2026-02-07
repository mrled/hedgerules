# Hedgerules

Hedgerules (from Hugo Edge Rules) is designed to work with `hugo deploy` and AWS CloudFront.

When deploying to CloudFront with S3 Static Website Hosting,
you have to implement custom headers and redirects yourself.
If you want to use CloudFront with S3 Object Storage API,
you also have to implement directory indices yourself
so that `/path` redirects to `/path/` and returns `/path/index.html` out of S3.
These are things bundled with services like Netlify.

Hedgerules exists to implement these features with platform-native objects
like CloudFront Functions and CloudFront Key Value Store (KVS).
Any path in a CloudFront Distribution can be handled by
at most one Viewer Request Function and at most one Viewer Response Function,
each of which can get data from a KVS.
Hedgerules sets function logic to implement headers, redirects, and indices,
and reads your built Hugo site to generate KVS items for each.

This lets you deploy a Hugo + Hedgerules site like this:

- Use `hedgerules deploy` to create the CloudFront Function and KVS for viewer request/response.
- Create an S3 bucket and CloudFront Distribution out of band.
  The CloudFront Distribution keeps a reference to the S3 bucket
  and the Functions.
- Use `hugo deploy` like normal to publish the site to the S3 bucket
  and invalidate the CloudFront cache.

We also include examples for:

- A Hugo theme with a template for generating a Hedgerules redirect JSON file.
- A CloudFormation template that deploys the S3 bucket and the CloudFront Distribution.
  It also includes log analysis with Athena.

Future ideas:

- Implement support for other clouds, particularly Azure and GCP since Hugo Deploy supports those.
