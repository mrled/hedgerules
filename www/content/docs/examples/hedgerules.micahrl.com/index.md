---
title: hedgerules.micahrl.com
---

This is how the <https://hedgerules.micahrl.com> documentation site is deployed.

These resources live in a dedicated AWS sub-account.
The DNS zone for micahrl.com is in a separate account;
DNS records are created out of band.

See [Deployment]({{< ref deployment >}})
for background on the stage model.

Configuration files:

- Hugo configs
  - [`_default/hugo.toml`](https://github.com/mrled/hedgerules/blob/master/www/config/_default/hugo.toml)
  - [`production/hugo.toml`](https://github.com/mrled/hedgerules/blob/master/www/config/production/hugo.toml)
    (overrides for the production environment)
- [`hedgerules.toml`](https://github.com/mrled/hedgerules/blob/master/hedgerules.toml)
- [`GNUmakefile`](https://github.com/mrled/hedgerules/blob/master/GNUmakefile)

See [Demonstration]({{< relref "/docs/demonstration" >}}) for examples of Hedgerules features on this site.

## Prerequisites

- AWS CLI configured with credentials for the site sub-account
- All stacks must be deployed in **us-east-1**
  (required for ACM certificates used by CloudFront)
- Access to the DNS account or provider to create records for micahrl.com

## Initial deployment

In the initial deployment, we have to handle some things manually related to DNS,
so this is broken into 5 stages.

### Stage 1: base infrastructure

Deploy [`base.cfn.yml`](https://github.com/mrled/hedgerules/blob/master/infra/prod/base.cfn.yml).
This creates the S3 bucket, OAC, ACM certificate, KVS stores, and IAM group.

```sh
make cfn-base
```

On the first deployment,
the stack will pause waiting for ACM certificate validation.
In a second terminal, find the validation CNAME:

```sh
# Get the validation CNAME record
aws acm describe-certificate \
    --region us-east-1 \
    --certificate-arn "$(aws acm list-certificates --region us-east-1 --query "CertificateSummaryList[?DomainName=='hedgerules.micahrl.com'].CertificateArn" --output text)" \
    --query 'Certificate.DomainValidationOptions[0].ResourceRecord'
```

Switch to the DNS account (or your external DNS provider)
and create the CNAME record returned above.
Once DNS propagates, ACM validates the certificate
and the CloudFormation stack completes.

Once complete, `make` saves the stack outputs to `dist/base.outputs.json`.
These are used automatically by the stage 4 target.
It also displays these values.
Copy the ContentBucketName into the Hugo configuration for `hugo deploy`.

### Stage 2: build

```sh
make hugo-build
```

This calls `hugo` with the appropriate configuration.

### Stage 3: hedgerules deploy

```sh
make hedgerules-deploy
```

On the first run this creates the CloudFront Functions.
On subsequent runs it updates their code and syncs KVS entries.

### Stage 4: distribution

Deploy [`distribution.cfn.yml`](https://github.com/mrled/hedgerules/blob/master/infra/prod/distribution.cfn.yml).
The make target reads stage 1 outputs from `dist/base.outputs.json`
and passes them as parameters automatically.

```sh
make cfn-distribution
```

If `base.cfn.yml` has changed since the last deploy,
`make` will re-deploy the base stack first.

It will show output like this:

```json
{
  "WebsiteURL": "https://hedgerules.micahrl.com",
  "DistributionId": "ET01CEXG1S3P9",
  "DistributionDomainName": "d447g1zqgagy6.cloudfront.net"
}
```

Switch to the DNS account and CNAME `hedgerules.micahrl.com`
to the distribution domain name (`d447g1zqgagy6.cloudfront.net` in this case).

The `DistributionId` value is also in the outputs file.
You need it for the `cloudFrontDistributionID` setting in `hugo.toml`.

### Stage 5: upload content

```sh
make hugo-deploy
```

## Subsequent deployments

After the initial deployment,
we can use a single make target to handle all stages:

```sh
make deploy
```

Unless something about the AWS resources needs to change,
only stages 2, 3, and 5 need to be run.
You could invoke this explicitly to avoid regenerating the CloudFormation outputs if you want
(maybe useful in CI):

```sh
make hugo-build hedgerules-deploy hugo-deploy
```

## CI/CD

Hedgerules uses GitHub actions to deploy the AWS infrastructure and the site content.

### IAM setup

To enable this, there is a third CloudFormation template that provisions an IAM group.
Members of this group can deploy CloudFormation stacks and run `make deploy`.

This stack is deployed once, manually.

**Template:** [`infra/ci/ci.cfn.yml`](https://github.com/mrled/hedgerules/blob/master/infra/ci/ci.cfn.yml)

Deploy it with:

```sh
aws cloudformation deploy \
  --region us-east-1 \
  --template-file infra/ci/ci.cfn.yml \
  --stack-name hedgerules-ci \
  --capabilities CAPABILITY_NAMED_IAM
```

Then, create an IAM user, add it to the group, and generate its access keys.

```sh
# Create the user and add to the CI group
aws iam create-user --user-name hedgerules-ci | cat
aws iam add-user-to-group \
  --user-name hedgerules-ci \
  --group-name hedgerules-ci-ci

# Generate access keys
aws iam create-access-key --user-name hedgerules-ci
```

Note that these access keys are shown only once, so make sure to save them for the next step.

### GitHub Actions

Add the `AccessKeyId` and `SecretAccessKey` from the output as repository secrets
`AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` in your GitHub Actions settings.

The workflow that deploys this site is found here:

- [`.github/workflows/deploy.yml`](https://github.com/mrled/hedgerules/blob/master/.github/workflows/deploy.yml)

It isn't related to deploying the site,
but for completeness we can also link to the workflow that builds the Go binary and uploads it to GitHub releases:

- [`.github/workflows/build.yml`](https://github.com/mrled/hedgerules/blob/master/.github/workflows/build.yml)
