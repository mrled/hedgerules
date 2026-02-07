// This file is embedded into the hedgerules binary and deployed to CloudFront.
// At deploy time, `var kvsId = '<arn>';` is prepended to this source.
// The CloudFront Functions runtime requires the KVS ID to be passed explicitly
// to cf.kvs() — there is no way to auto-discover an associated KVS.

import cf from 'cloudfront';

async function handler(event) {
  var request = event.request;
  var uri = request.uri;

  // Single KVS lookup for redirect
  try {
    var kvs = cf.kvs(kvsId);
    var dest = await kvs.get(uri);
    if (dest) {
      return {
        statusCode: 301,
        statusDescription: 'Moved Permanently',
        headers: { 'location': { value: dest } }
      };
    }
  } catch (err) {
    // Key not found, continue to index rewrite
  }

  // Append index.html for directory requests
  if (uri.endsWith('/')) {
    request.uri += 'index.html';
  }

  return request;
}
