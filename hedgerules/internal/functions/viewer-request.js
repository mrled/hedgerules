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
