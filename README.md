# Hugo EdgeRules

This project is designed to work with `hugo deploy` to static hosting with edge rules, like CloudFront.

- A Hugo theme that includes a template for generating a JSON file with headers.
- a binary that is designed to update cloud front key value store with lists of all redirects, including redirecting a directory path of out trailing trailing slash one with a trailing slash (`/path` -> `/path/`)
- deploy a viewer request function that handles redirects and directories to index.html (request for `/path/`, return `/path/index.html`)
- deploy a viewer response function that injects custom headers (like Netlify's `_headers` file)

It expects you to create the requisite cloud resources (CloudFront Distribution, Function, KVS, etc) in advance yourself.