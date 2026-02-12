---
title: "KVS Limitations"
weight: 40
---

# KVS Limitations

CloudFront KeyValueStore (KVS) is central to how hedgerules works -- redirects and headers are stored in KVS and read by CloudFront Functions at the edge. KVS is fast and cost-effective, but it has hard constraints that shape what hedgerules can and cannot do.

## Constraints

<table>
  <tr>
    <th>Constraint</th>
    <th>Limit</th>
    <th>Notes</th>
  </tr>
  <tr>
    <td>Max key length</td>
    <td>512 bytes</td>
    <td></td>
  </tr>
  <tr>
    <td>Max key + value size</td>
    <td>1 KB</td>
    <td>Redirect source paths + destination URLs must fit in 1 KB combined. The same applies to custom response header values.</td>
  </tr>
  <tr>
    <td>Total KVS size</td>
    <td>5 MB</td>
    <td>Generous for most sites, but redirects and custom header rules share this capacity. Can become relevant at scale.</td>
  </tr>
  <tr>
    <td>KVS stores per function</td>
    <td>1</td>
    <td>All data for a given function must share a single store, using key prefixes or conventions to distinguish entry types.</td>
  </tr>
  <tr>
    <td>KVS access from functions</td>
    <td>Read-only</td>
    <td>Data can only be written via the AWS API (<code>hedgerules deploy</code>). No counters, rate tracking, or stateful logic at the edge.</td>
  </tr>
  <tr>
    <td>Enumeration from functions</td>
    <td>Not supported (no <code>list()</code> or prefix scan)</td>
    <td>Every lookup must be by exact key. No pattern matching or wildcard lookups at runtime.</td>
  </tr>
</table>

## KVS and blocking

You could implement blocking via KVS, but the limitations above would make this very limited.
Some specific concerns:
- User-agent blocking requires substring matching against many patterns. KVS has no `list()` operation, so the function cannot iterate over stored patterns. Workarounds like packing patterns into a single value hit the 1 KB value limit.
- IP range (CIDR) blocking requires bitwise math that is difficult to fit within the 1ms CloudFront Function execution budget.
- Stacking multiple blocking checks (IP + country + user-agent + path + method) on top of the existing redirect/header logic risks exceeding the compute budget.
- The 10KB code size limit is also a factor

AWS WAF is better suited for this, but does cost money.

CloudFront geographic restrictions are a free option for country-level blocking that applies to the whole site.

