---
name: testx
description: Write, review, repair, or explain Go tests that use github.com/go-testx/testx. Use for testx assertions, typed Cases, Run/RunErr and presets, interface contracts, Eventually polling, HTTP/CLI tests, JSON/golden comparisons, benchmarks, fuzz seeds, migration from standard or other assertion libraries, and diagnosing generated testx code.
---

# testx Go Testing

Use testx as a layer over Go's standard `testing` package. Preserve standard test functions and fall back to `testing.T` whenever the declarative API does not fit.

## Workflow

1. Inspect the subject signature, observable behavior, related types, existing tests, and project conventions.
2. Read [references/overview.md](references/overview.md) for selection rules and core semantics.
3. Read only the references needed for the task:
   - [references/api-reference.md](references/api-reference.md) before using an unfamiliar API or checking exact signatures.
   - [references/recipes.md](references/recipes.md) when generating a specific kind of test.
   - [references/constraints.md](references/constraints.md) when reviewing, repairing, or substantially rewriting tests.
4. Choose the lowest testx abstraction that removes real repetition:
   - assertions plus standard `testing` for custom flow;
   - `Case` plus hand-written `t.Run` for custom per-Case logic;
   - `Run`, `RunErr`, or a preset only for an exact supported signature;
   - `Contract` for behavior shared by interface implementations.
5. Derive expected values from code, requirements, or existing behavior. Do not turn invented placeholders into assertions.
6. Format the result and run the narrow test, then `go test ./...` when the workspace permits.

## Non-negotiable rules

- Keep `func TestXxx(t *testing.T)` and the standard Go test runner.
- Use `Require` for prerequisites and `Assert` for independent checks.
- Do not generate `if !testx.Assert(...){ t.Fatal(...) }`; assertions already report failures.
- Remember that `RunErr` expects `nil` error by default.
- Add `.WithError(...)` only to expected-error Cases.
- Do not invent testx identifiers. Use standard `testing` when the reference has no matching API.
- Keep tests deterministic and bound timeouts, processes, polling, and external resources.
