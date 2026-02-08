---
title: To do
weight: 99
---

Future plans, maybe with prompts

## Undo incorrect architectural decisions

Some of the [decisions](/docs/development/decisions) made were too minimalist.

## Add infra to deploy to AWS

Currently the site is being deployed to GitHub Pages. I want to move it to CloudFront. Look at the docs in docs/ and also the www/ site content to see what AWS objects must be created outside of Hedgerules. You can also see some more complex examples (dual sites) in examples/. Make a new folder, infra/prod/, which contains resources required to host the whole site. Use CloudFormation templates. Do the bare minimum for as simple as possible, but make sure it has enough to deploy the site. It will use hedgerules.micahrl.com as its domain name, but DNS will be handled by an external provider - do not create DNS records. i do need HTTPS  certificates though. Will this be possible?

## Support more clouds

- Azure
- GCP
- CloudFlare
