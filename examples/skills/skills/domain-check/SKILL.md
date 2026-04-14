---
name: check-domain-availability
description: Check whether a domain is reachable using available network tools.
---

## Goal

Determine whether a domain is online and explain the evidence briefly.

## Steps

1. If a web/tool capability is available, query the domain with a lightweight request.
2. Report whether the domain appears online based on status code, DNS resolution, or fetch result.
3. If the check fails, include likely reasons (DNS issue, timeout, blocked network).

## Output

- One short conclusion line.
- One line with key evidence.
