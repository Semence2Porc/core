# Terra Classic (`classic-terra/core`) Fund-Safety Audit — Findings

**Date:** 2026-09-09 · **Scope:** `main` @ `9a5ee56` (plus PR #664 head `c5bf7ed` reviewed separately)
**Method:** manual code review of every fund-moving path, with each finding proven by a regression test that fails on the vulnerable code; all fixes verified in a Linux container (golang:1.24) and with the repo's pinned golangci-lint v2.1.6.

## Findings shipped as PRs

| # | Severity | Finding | PR |
|---|----------|---------|-----|
| 1 | **Chain-halt via governance** | `x/tax/keeper/tax_split.go` — `ProcessTaxSplits` divides by `communityTax·(1−oracleSplitRate)`. With `communityTax = 1.0` (allowed by SDK distribution param validation) and `oracleSplitRate = 0` (allowed by `validateOraceSplit`), the denominator is zero and `LegacyDec.Quo` **panics inside every taxable transaction** (bank sends, market swaps, wasm calls via the tax handlers). One passed governance proposal → chain halt until emergency upgrade. Reproduced: `PANIC-CONFIRMED: division by zero` on the exact original expression. | [#670](https://github.com/classic-terra/core/pull/670) |
| 2 | **Node/tool panic (latent)** | `x/tax/keeper/keeper.go` — `IsReverseCharge` performed an unchecked type assertion on the reverse-charge context flag; any context that bypassed the ante handler (queries, genesis import/export, external tools) panicked on `nil`. | [#667](https://github.com/classic-terra/core/pull/667) |
| 3 | **Operational blindness** | `x/oracle/abci.go` — EndBlocker silently skipped the entire vote period (no tally, no miss counting, no rewards, zero logs) when staking-keeper calls failed. | [#668](https://github.com/classic-terra/core/pull/668) |
| 4 | **Missing disclosure channel** | No SECURITY.md; GitHub private vulnerability reporting disabled — no documented private path for reporting fund-affecting bugs. | [#669](https://github.com/classic-terra/core/pull/669) |
| 5 | **Broken test infrastructure** | interchaintest PFM tests pull `ghcr.io/strangelove-ventures/heighliner/osmosis`, dead since the org moved to `amygdala-labs` — every PR's e2e check red since then (last green 2026-07-27). Fixed via ChainSpec image override; **CI re-ran the real PFM test: pass (13m)**. | #667 (commit e718e86) |

## Paths reviewed and found sound

- **Market swap custody flow** (`x/market/keeper/msg_server.go`): burn-offer → mint-ask sequence is atomic under the SDK's error-tainted context; conservation holds (`mintCoins = swapCoin + feeCoin`; spread never negative; zero-output rejected by `ErrZeroSwapCoin`); extreme-amount guard (`BitLen() > 100`) present in simulation path; fees routed to the oracle account per spec.
- **Constant-product math** (`swap.go`): pools derived from `basePool²`; no division by user-controlled zero (divisors are pool quantities bounded above zero by construction).
- **Tax computation** (`x/tax/types/compute.go`): bond denom and IBC denoms correctly excluded; per-denom tax caps applied; rounding truncates (no over-collection).
- **Tax-exemption governance authority**: every message in `x/taxexemption/keeper/msg_server.go` enforces `GetAuthority() == msg.Authority` — exemptions cannot be self-granted.
- **Epoch burn** (`x/treasury/keeper/burn_account.go`): burns everything in the burn module account; error panics (fail-loud inside EndBlock, correct for consensus).
- **PR #664 (Market Module 2.0)**: previous deep review found no fund-stealing issue; improvements suggested as review comments (TWAP time-weighting, swap telemetry events for silent TWAP-deviation/daily-cap rejections).

## Residual observations (not fixed in this pass)

- `getTxPriority` was re-checked and **already implements the hardened upstream pattern** (`IsInt64()` clamp + min-across-coins) — earlier note withdrawn.
- gofmt/gci flag ~20 pre-existing `x/tax` files on a CRLF Windows checkout; the canonical LF tree lints **0 issues** — cosmetic only, no PR needed.
- Governance parameter bounds: consider validating `communityTax < 1.0`-adjacent combinations at the treasury/distribution param layer as defense-in-depth on top of #670's runtime guard.

## Part 2 — oracle price-staleness windows & treasury distribution params

### Oracle

- **Staleness safety property confirmed.** Rates are *deleted* at the start of every vote period and only re-set when quorum is reached (`PickReferenceTerra` non-empty in `x/oracle/abci.go`). A dead oracle leaves **no** rates behind — swaps fail closed. No stale-rate window exists.
- **Tally / median / cross-rate edge cases are guarded.** `ToCrossRate`/`ToCrossRateWithSort` convert voters without a positive reference rate to zero-power abstains (no division by zero); `WeightedMedian`/`StandardDeviation` handle empty ballots; abstain voters are excluded from rewards but not slashed (documented classic-Terra semantics).
- **Slash math is safe.** `validVoteRate = (periods − misses)/periods`; even the uint64-underflow case (misses > periods, possible after a mid-window `SlashWindow` decrease) yields a negative rate that still slashes correctly.
- **Reward pool is bank-backed.** `GetRewardPool` reads the live module balance (`alias_functions.go`), not a stored counter — the `SendCoinsFromModuleToModule` panic in `RewardBallotWinners` cannot be triggered by a bookkeeping divergence.
- Net: **no new finding** beyond #668 (already shipped). The oracle module's remaining risks are governance-param choices, not code paths.

### Treasury

- **Policy math is guarded.** `UpdateTaxPolicy`/`UpdateRewardPolicy` special-case zero denominators (return `RateMax`); `Clamp` bounds every result; `TRL` protects the `tr/tsl` division.
- **Indicators are best-effort.** `alignCoins` skips unpriceable denoms; a `TotalBondedTokens` error degrades TSL to zero rather than panicking (the oracle-`#668` pattern here degrades gracefully — no PR warranted).
- **Authority gates hold.** Burn-tax-exemption mutations only via governance proposals (`gov.go`); seigniorage settlement is **dead code** (commented out in the EndBlocker, `TODO`); params are validated for rate ordering, negatives, and window relations.
- Net: **no new finding**; the only treasury-adjacent issue (community-tax div-by-zero) was #670.

### PR #664 (MM2) — review posted

Deep review of head `c5bf7ed` (pool-based swaps, daily caps, TWAP, oracle hook, tax redirect). Fund paths conserve value; **no theft or chain-halt path found**. Review [posted](https://github.com/classic-terra/core/pull/664#pullrequestreview-5159786710) with six points:

1. **`SwapFeeBurnRate + SwapFeeCommunityRate` not jointly validated** (each [0,1] via `validateFraction`, sum unbounded) — a governance proposal setting both to 1.0 makes every swap fail at `FundCommunityPool` after the burn. Same governance-DoS class as #670. Fix: cross-param check in `Params.Validate()`.
2. `ComputeTWAP` is a simple average, not time-weighted (code comment agrees) — snapshots underweight failed-quorum stretches; weight by stored `Height` intervals.
3. Swap-rejection paths (`ErrTWAPDeviation` / `ErrDailyCapExceeded` / `ErrOraclePriceStale`) are silent — no events/telemetry, unlike the new tax-split events.
4. `ResetDailyCapIfNeeded` treats 14,400 blocks as a day (3s assumption drifts with real block time).
5. Rebase coordination with #670 (same `tax_split.go` function) and #668 (same oracle EndBlocker region).
6. Minor: constant-product divisor `offerPool + offerAmt` can reach zero with a deeply negative `TerraPoolDelta` (pre-existing; defensive guard recommended).

## Verification standard used

For every change: CI-parity Linux container (`go build ./...`, `go vet`, full module test suites, gofmt, pinned golangci-lint v2.1.6 on the canonical LF tree), plus fail-on-vulnerable-code proof (stash fix → test fails → restore fix → test passes). The interchain e2e claim is backed by an actual CI run of `test-ibc-pfm` on PR #667 — not a local approximation.

## CI lessons captured during this work

- **`LegacyDec` rounds per operation.** The first #670 helper computed `(ct·osr)/den` instead of the original `ct·(osr/den)` — mathematically equal, different in the last decimals. Main's own ante suite caught it in CI; the shipped helper preserves the exact original operation order. Existing test coverage is what makes this fix safe.
- **Canonical-tree linting.** A Windows CRLF checkout makes gci/gofumpt flag ~20 untouched files; linting the `git archive` (LF) tree proved the committed content clean and isolated real issues (two in our own new files — both fixed before push).
- **Repo-wide CI rot.** The `strangelove-ventures` → `amygdala-labs` org rename broke the interchain e2e for *every* PR since late July; the per-PR image override (now on all four branches) restores it until interchaintest updates its built-in config.
