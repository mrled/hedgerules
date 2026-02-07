// This file is embedded into the hedgerules binary and deployed to CloudFront.
// At deploy time, `var kvsId = '<arn>';` is prepended to this source.
// The CloudFront Functions runtime requires the KVS ID to be passed explicitly
// to cf.kvs() — there is no way to auto-discover an associated KVS.

import cf from 'cloudfront';

async function handler(event) {
  var response = event.response;
  var request = event.request;
  var kvs = cf.kvs(kvsId);
  var path = request.uri;
  var headers = {};

  // Root fallback headers (lowest priority)
  try {
    var rootValue = await kvs.get('/');
    if (rootValue) {
      var lines = rootValue.split('\n');
      for (var i = 0; i < lines.length; i++) {
        var line = lines[i].trim();
        var idx = line.indexOf(':');
        if (idx !== -1) {
          headers[line.substring(0, idx).trim().toLowerCase()] = line.substring(idx + 1).trim();
        }
      }
    }
  } catch (err) {}

  // Exact path headers (highest priority, overrides root)
  if (path !== '/') {
    try {
      var pathValue = await kvs.get(path);
      if (pathValue) {
        var lines = pathValue.split('\n');
        for (var i = 0; i < lines.length; i++) {
          var line = lines[i].trim();
          var idx = line.indexOf(':');
          if (idx !== -1) {
            headers[line.substring(0, idx).trim().toLowerCase()] = line.substring(idx + 1).trim();
          }
        }
      }
    } catch (err) {}
  }

  // Apply collected headers to response
  var names = Object.keys(headers);
  for (var i = 0; i < names.length; i++) {
    response.headers[names[i]] = { value: headers[names[i]] };
  }

  return response;
}
