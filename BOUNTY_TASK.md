# Bounty Fix

**Reward:** $10.00
**Issue:** https://github.com/NashKara96/fx/issues/1
**Repo:** https://github.com/NashKara96/fx

---

## Issue

🎯 Ensure all OnStop hooks execute during Lifecycle.Stop() even if a prior hook returns context.Canceled or context.DeadlineExceeded

## 📝 Description
In the current lifecycle management implementation, when `Lifecycle.Stop()` is invoked, it executes the registered `OnStop` hooks in reverse order of their registration. However, if any of these hooks returns an error—specifically `context.Canceled` or `context.DeadlineExceeded`—the execution loop aborts immediately. 

This premature termination prevents subsequent shutdown hooks from running. As a result, critical resources registered later in the lifecycle (such as network listeners, background workers, metrics exporters, or database connection pools) are never gracefully closed, leading to resource leaks, dangling connections, and unclean application shutdowns.

While the failing hook should still report its error, the lifecycle manager must make a best-effort attempt to execute all remaining cleanup callbacks.

## 🎯 Acceptance Criteria
- [ ] `Lifecycle.Stop()` must not abort the execution of remaining `OnStop` hooks when a hook returns an error, including `context.Canceled` or `context.DeadlineExceeded`.
- [ ] All registered `OnStop` hooks must be attempted during shutdown.
- [ ] Errors encountered during the execution of `OnStop` hooks must be accumulated (e.g., using `go.uber.org/multierr` or a similar error aggregation mechanism) and returned as a single combined error after all hooks have been executed.
- [ ] The reverse-order execution of `OnStop` hooks must be strictly preserved.
- [ ] A timeout or cancellation of the context passed to `Stop(ctx)` should still be respected, but individual hook failures must not artificially trigger a halt for the remaining hooks.

## 🛠️ Technical Specifications & Context
- **Target Files:** The lifecycle execution logic is likely located in `lifecycle.go`, `app.go`, or an internal runner file within the `NashKara96/fx` repository (modeled after `uber-go/fx`).
- **Hook Execution Loop:** Locate the loop in `Lifecycle.Stop()` (or the internal runner executing the stop hooks). It currently looks similar to:
  ```go
  for i := len(lifecycle.position) - 1; i >= 0; i-- {
      // ... execution logic ...
      if err := hook.Stop(ctx); err != nil {
          return err // ❌ Premature exit
      }
  }
  ```
- **Proposed Fix:** Modify the loop to accumulate errors instead of returning early:
  ```go
  var errs error
  for i := len(lifecycle.position) - 1; i >= 0; i-- {
      // ... execution logic ...
      if err := hook.Stop(ctx); err != nil {
          errs = multierr.Append(errs, err) // Accumulate error and continue
      }
  }
  return errs
  ```

## 🧪 Verification & Testing
- **Unit Tests:** Add a test case in `lifecycle_test.go` (or equivalent test suite) that:
  1. Registers three dummy components with `OnStop` hooks.
  2. Configures the second hook (in execution order) to return `context.Canceled` or `context.DeadlineExceeded`.
  3. Asserts that the third hook is still executed despite the failure of the second.
  4. Asserts that `Lifecycle.Stop()` returns an aggregated error containing the failure from the second hook.
- **Integration Test:** Verify that when the application receives a shutdown signal and a component fails to stop within its allocated time, other components still release their file descriptors and network ports.

---

[![Opire Bounty](https://img.shields.io/badge/bounty-Opire-emerald?style=for-the-badge)](https://github.com/NashKara96/fx)
<br/>
<hr/>

<details><summary>This repo is using Opire - what does it mean? 👇</summary><br/>💵 Everyone can add rewards for this issue commenting <code>/reward 100</code> (replace <code>100</code> with the amount).<br/>🕵️‍♂️ If someone starts working on this issue to earn the rewards, they can comment <code>/try</code> to let everyone know!<br/>🙌 And when they open the PR, they can comment <code>/claim #1</code> either in the PR description or in a PR's comment.<br/><br/>🪙 Also, everyone can tip any user commenting <code>/tip 20 @NashKara96</code> (replace <code>20</code> with the amount, and <code>@NashKara96</code> with the user to tip).<br/><br/>📖 If you want to learn more, check out our <a href="https://docs.opire.dev">documentation</a>.</details>

---

## Process

### 1. Orient yourself first
Before touching any code:
- Read the README and any relevant docs
- Run: find . -type f | grep -v node_modules | grep -v .git | grep -v dist | head -60
- Grep for the key terms from the issue title to find the relevant files
- Read the files around the bug — understand what's currently happening

### 2. Find and read the tests
- Look for test files near the relevant code (*.test.ts, *.spec.js, __tests__, etc.)
- Read them — they tell you what behavior is expected
- Understand what a passing fix looks like before writing a line

### 3. Implement the fix
- Make the MINIMAL change that fixes the issue — nothing more
- Do not refactor, rename, reorganize, or clean up unrelated code
- Do not add dependencies unless the issue explicitly requires it
- Do not modify config files (package.json, tsconfig.json, etc.) unless that IS the fix
- One logical change, one commit

### 4. Verify
- Run the test suite: check package.json scripts for "test", "lint", "typecheck"
  Common commands: npm test, yarn test, pnpm test, pytest, go test ./..., cargo test
- If TypeScript is present: npx tsc --noEmit
- Fix any test failures your change caused — if a test was wrong before, update it

### 5. Commit and document
- git add -A
- git commit -m "fix: 🎯 Ensure all OnStop hooks execute during Lifecycle.Stop() even if a pri"
- Write AGENT_SUMMARY.md with:
  - Root cause: what was wrong and why
  - Fix: what you changed and the reasoning
  - Verification: how to confirm it works
  - Concerns: anything a reviewer should know

---

## Stop conditions

If ANY of these are true, write "UNCERTAIN: [reason]" at the top of AGENT_SUMMARY.md, make NO code changes, and stop:
- The fix requires changing more than ~5 files
- You cannot determine what the correct behavior should be
- The issue requires a database migration, API key, or external service
- You find 3+ open PRs already attempting this fix (high competition)
- The repo is not in a language/framework you can reason about confidently

A clean UNCERTAIN is better than a bad PR.
