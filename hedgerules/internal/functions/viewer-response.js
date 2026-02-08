// This file is embedded into the hedgerules binary and deployed to CloudFront.
// At deploy time, `var kvsId = '<arn>';` and `var debugHeaders = true/false;`
// are prepended to this source.

import cf from 'cloudfront';

async function handler(event) {
  var response = event.response;
  var request = event.request;
  var kvs = cf.kvs(kvsId);
  var path = request.uri;
  if (path.charAt(0) !== '/') {
    path = '/' + path;
  }

  try {
    // Build list of patterns to check in order of specificity (least to most).
    // Later matches override earlier ones.
    var patterns = [];

    // 1. Root path (always check, lowest priority)
    patterns.push('/');

    // 2. Each parent directory with trailing /
    var parts = path.split('/').filter(function(p) { return p; });
    for (var i = 0; i < parts.length - 1; i++) {
      patterns.push('/' + parts.slice(0, i + 1).join('/') + '/');
    }

    // 3. Extension wildcard (e.g., *.xml, *.html)
    var lastSegment = parts[parts.length - 1] || '';
    var lastDotIndex = lastSegment.lastIndexOf('.');
    if (lastDotIndex !== -1 && lastDotIndex < lastSegment.length - 1) {
      patterns.push('*.' + lastSegment.substring(lastDotIndex + 1));
    }

    // 4. Exact path (most specific)
    if (path !== '/') {
      patterns.push(path);
    }

    // Collect headers from all matching patterns
    var headers = {};
    var matched = [];

    for (var i = 0; i < patterns.length; i++) {
      try {
        var value = await kvs.get(patterns[i]);
        if (value) {
          matched.push(i);
          var lines = value.split('\n');
          for (var j = 0; j < lines.length; j++) {
            var line = lines[j].trim();
            var idx = line.indexOf(':');
            if (idx !== -1) {
              var name = line.substring(0, idx).trim().toLowerCase();
              var val = line.substring(idx + 1).trim();
              val = val.replace('{/path}', path);
              if (name) {
                headers[name] = val;
              }
            }
          }
        }
      } catch (err) {
        // Key not found in KVS, continue
      }
    }

    // Apply collected headers to response
    var names = Object.keys(headers);
    for (var i = 0; i < names.length; i++) {
      response.headers[names[i]] = { value: headers[names[i]] };
    }

    // Debug headers (conditional on injected debugHeaders variable)
    if (typeof debugHeaders !== 'undefined' && debugHeaders) {
      response.headers['x-hedgerules-patterns'] = { value: patterns.join(',').substring(0, 200) };
      response.headers['x-hedgerules-matched'] = { value: matched.join(',').substring(0, 200) };
    }

  } catch (err) {
    // Critical error — add error debug header (always emitted) and return response unchanged
    var msg = '';
    if (err && err.message) {
      msg = err.message;
    } else if (err) {
      msg = String(err);
    }
    response.headers['x-hedgerules-error'] = { value: msg.substring(0, 200).replace(/[\r\n]+/g, ' ') };
  }

  return response;
}
