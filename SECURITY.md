# Security Policy

## Supported versions

Security fixes are applied to the latest released version of `classic-terra/core` and to the `main` branch.

## Reporting a vulnerability

**Please do not open a public issue for anything that could put user funds, chain liveness, or validator keys at risk.**

Preferred disclosure path:

1. Use GitHub's **private vulnerability reporting** on this repository (Security tab → "Report a vulnerability"). If the feature is not yet enabled, please ask a repository maintainer to enable it, or contact a maintainer directly to arrange a private channel.
2. Include in your report:
   - The affected version or commit hash.
   - Reproduction steps or a failing test case (a Go test is ideal).
   - Your assessment of the impact (funds at risk, consensus divergence, DoS, etc.).
   - A suggested fix, if you have one.

Non-sensitive issues (typos, internal errors with no security impact, missing logs) can go through regular public issues and pull requests.

## What to expect

- Acknowledgement of your report within 72 hours.
- A coordinated disclosure timeline agreed with you before any public detail is published.
- Public credit for the discovery unless you prefer to remain anonymous.
- Critical fixes ship as a priority release, coordinated with validators so they can upgrade promptly.

## Scope

**In scope:**

- This repository: chain core modules (`x/market`, `x/oracle`, `x/tax`, `x/treasury`, `x/taxexemption`, `x/dyncomm`, ...), ante decorators, wasm bindings, and the node/CLI tooling.
- Consensus-affecting bugs (divergence, halts, non-determinism).
- Fund-affecting bugs (loss, theft, mint/burn accounting, fee or tax handling).
- Genesis and upgrade-handler logic.

**Out of scope:**

- Third-party applications, frontends, or wallets not hosted in this organization.
- Social engineering of validators, maintainers, or community members.
- Denial of service that relies solely on exhausting resources of publicly reachable nodes (rate limits).

## Bug bounty

There is currently no formal bounty program for this repository. Reports of high-impact vulnerabilities are still welcomed and will be credited.
