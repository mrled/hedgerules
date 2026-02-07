# Hedgerules

Hedgerules (from Hugo Edge Rules) is designed to work with `hugo deploy` and AWS CloudFront.

When deploying to CloudFront with S3 Static Website Hosting,
you have to implement custom headers and redirects yourself.
If you want to use CloudFront with S3 Object Storage API,
you also have to implement directory indices yourself
so that `/path` redirects to `/path/` and returns `/path/index.html` out of S3.
These are things bundled with services like Netlify.

(TODO: Add notes on why you might want to use CloudFront with S3 Object Storage API
instaed of S3 Static Website Hosting.)

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
- A Hugo site with documentation for Hedgerules.
  Aside from the docs, this site also serves as an example site for users to reference,
  and contains paths we can use for testing.

Tests:

- Test header injection based on exact path, parent path, and wildcard
- Test header injection with the special `{/path}` token
- Test redirects from Hugo aliases
- Test user-defined redirects
- Test index redirects (`/path` -> `/path/`)
- Test index responses (A request for `/path/` should return the S3 object at `/path/index.html`)

Future ideas:

- Implement support for other clouds, particularly Azure and GCP since Hugo Deploy supports those.
- Have Hedgerules parse the Hugo config file for special site parameters
  so the user can define the Function and KVS object names in one place,
  next to `hugo deploy` settings.
