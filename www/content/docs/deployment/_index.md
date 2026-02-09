---
title: "Deployment"
weight: 45
---

Deploying a Hedgerules site requires five stages.
Stages 1 and 4 are initial setup (CloudFormation);
stages 2, 3, and 5 run on every content update.

<table>
  <thead>
    <tr>
      <th>Stage</th>
      <th>Object</th>
      <th>Notes</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td rowspan="7"><strong>1. CloudFormation stack</strong></td>
      <td>S3 content bucket</td>
      <td>Static file storage; no website hosting needed</td>
    </tr>
    <tr>
      <!--<td></td>-->
      <td>S3 bucket policy</td>
      <td>Grants CloudFront OAC read access</td>
    </tr>
    <tr>
      <!--<td></td>-->
      <td>Origin Access Control</td>
      <td>Grants CloudFront access to S3</td>
    </tr>
    <tr>
      <!--<td></td>-->
      <td>ACM Certificate</td>
      <td>DNS validation for custom domain</td>
    </tr>
    <tr>
      <!--<td></td>-->
      <td>CloudFront KVS: redirects</td>
      <td>Empty store; entries managed in stage 3</td>
    </tr>
    <tr>
      <!--<td></td>-->
      <td>CloudFront KVS: headers</td>
      <td>Empty store; entries managed in stage 3</td>
    </tr>
    <tr>
      <!--<td></td>-->
      <td>IAM deployment group</td>
      <td>Permissions for S3, CloudFront, KVS, and Function APIs</td>
    </tr>
    <tr>
      <td rowspan="4"><strong>2. <code>hugo</code></strong></td>
      <td><code>_hedge_redirects.txt</code></td>
      <td>Alias redirects; <code>hedgeredirects</code> output format</td>
    </tr>
    <tr>
      <!--<td></td>-->
      <td><code>_hedge_headers.json</code></td>
      <td>Path&ndash;header map; <code>hedgeheaders</code> output format</td>
    </tr>
    <tr>
      <!--<td></td>-->
      <td>Directory structure</td>
      <td>Scanned by hedgerules to produce <code>/path</code> &rarr; <code>/path/</code> redirects</td>
    </tr>
    <tr>
      <!--<td></td>-->
      <td>Site files (HTML, CSS, JS, images)</td>
      <td>Used by stage 5</td>
    </tr>
    <tr>
      <td rowspan="4"><strong>3. <code>hedgerules deploy</code></strong></td>
      <td>Viewer-request function</td>
      <td>Created on first deploy, code updated on each subsequent deploy</td>
    </tr>
    <tr>
      <!--<td></td>-->
      <td>Viewer-response function</td>
      <td>Created on first deploy, code updated on each subsequent deploy</td>
    </tr>
    <tr>
      <!--<td></td>-->
      <td>Redirects KVS entries</td>
      <td>Convergent sync against KVS from stage 1</td>
    </tr>
    <tr>
      <!--<td></td>-->
      <td>Headers KVS entries</td>
      <td>Convergent sync against KVS from stage 1</td>
    </tr>
    <tr>
      <td rowspan="2"><strong>4. CloudFormation stack</strong></td>
      <td>CloudFront Distribution</td>
      <td>References S3 bucket from stage 1 and functions from stage 3</td>
    </tr>
    <tr>
      <!--<td></td>-->
      <td>Route53 DNS record</td>
      <td>Alias to distribution</td>
    </tr>
    <tr>
      <td rowspan="2"><strong>5. <code>hugo deploy</code></strong></td>
      <td>S3 objects</td>
      <td>Uploaded to bucket from stage 1</td>
    </tr>
    <tr>
      <!--<td></td>-->
      <td>CloudFront cache invalidation</td>
      <td>Invalidates distribution from stage 4</td>
    </tr>
  </tbody>
</table>

We recommend CloudFormation for stages 1 and 4,
but anything that can deploy resources to AWS works fine:
the AWS web console,
the `aws` commandline program,
Terraform, etc.
For the purposes of most of our docs,
we assume CloudFormation.

CloudFormation cannot own the CloudFront Functions because `hedgerules deploy` updates their code on every deploy,
which would cause stack drift.
The same is true for any other declarative approach like Terraform.
The KVS resources are safe in CloudFormation because CloudFormation does not track individual KVS entries.

The Distribution is in a separate stack (stage 4) because it references the function ARNs,
which don't exist until `hedgerules deploy` creates them in stage 3.

## Initial setup

```sh
# Stage 1: base infrastructure
aws cloudformation deploy \
  --template-file infra.cfn.yml \
  --stack-name mysite-infra \
  --capabilities CAPABILITY_NAMED_IAM

# Stage 2: build
hugo

# Stage 3: create functions + sync edge data
hedgerules deploy --site public/

# Stage 4: distribution (references functions from stage 3)
aws cloudformation deploy \
  --template-file distribution.cfn.yml \
  --stack-name mysite-distribution

# Stage 5: upload content
hugo deploy --target mysite
```

See [`examples/micahrlweb/`](https://github.com/micahrl/hedgerules/tree/master/examples/micahrlweb) for a complete CloudFormation template.

## Each content update

```sh
hugo                              # Stage 2
hedgerules deploy --site public/  # Stage 3
hugo deploy --target mysite       # Stage 5
```

The order matters:

1.  **`hugo`** must run first because it generates `_hedge_redirects.txt`, `_hedge_headers.json`,
    and the directory structure that `hedgerules deploy` reads.
2.  **`hedgerules deploy`** should probably run before `hugo deploy`
    so that redirects and headers are in place before new content goes live.
    This avoids a window where new pages are served without their headers
    or where old redirects point to not-yet-uploaded content.
3.  **`hugo deploy`** runs last, uploading content to S3 and invalidating the CloudFront cache.

Re-running `hedgerules deploy` is always safe.
The KVS sync is convergent: it computes a diff against the existing state and only applies changes.
