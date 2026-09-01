# Repository Engineering Requirements

These requirements apply to every agent and every change in this repository. Preserve unrelated user changes in the working tree.

## Evidence before optimization

- Never claim a performance improvement without a benchmark that exercises the changed production path through its real package API or the narrowest representative internal entry point.
- For every production-code change, identify the affected benchmark before editing. If none exists, add one. Include realistic common input and a relevant adverse input (large, binary, highly concurrent, cache miss, or error path as appropriate).
- Record a before and after result on the same machine, Go version, environment, and benchmark definition. Run at least five samples with allocation reporting, for example:

  ```sh
  go test ./path/to/pkg -run '^$' -bench '^BenchmarkName$' -benchmem -count=5
  ```

- Prefer `benchstat` when available. Otherwise report the median of the samples and disclose material variance or outliers. Keep raw results outside the repository unless the user requests an artifact.
- Do not accept a change that causes a credible regression in the common production workload or increases `B/op` or `allocs/op`, unless the user explicitly accepts a documented trade-off supported by end-to-end data.
- Microbenchmarks are decision evidence, not proof of production impact. Explain how the measured path is reached in production. Reject optimizations whose benefit exists only in test or benchmark code.
- Documentation-only, generated-output-only, and test-only changes do not require a new benchmark, but must not modify or weaken existing benchmark coverage.

## Correctness and compatibility gates

- Run the focused tests for each affected package while iterating.
- Before handing off any code change, run:

  ```sh
  go test ./...
  go test -race ./...
  go vet ./...
  git diff --check
  ```

  `make test` may replace the first two test commands because it already runs coverage and the race detector. Run repository generation checks when generated sources or descriptors could change.
- Do not change an exported name, type, function signature, interface contract, serialized representation, error identity, concurrency guarantee, or ownership behavior without explicitly identifying it as a breaking change and obtaining user approval.
- For algorithm rewrites, add a compatibility or differential test against the previous behavior. Cover boundary values exhaustively when the input domain is small; otherwise use table-driven edge cases plus fuzzing/property tests where useful.
- A benchmark win never overrides a correctness, compatibility, race, or ownership failure.

## Memory safety and ownership review

For every code change, inspect the changed data flow and test the applicable risks below:

- **Aliasing and ownership:** Determine who owns slices, maps, protobuf messages, descriptors, buffers, and returned views. Copy at API or goroutine boundaries when callers could mutate shared backing storage. Do not introduce unsafe string/byte aliasing.
- **Concurrency:** Protect shared mutable state, avoid copying live locks, and verify publication happens-before use. Exercise concurrent paths with the race detector.
- **Pooling and retention:** Reset pooled objects before reuse, never return them to a pool while references can escape, discard abnormally large backing arrays, and avoid retaining request payloads or secrets longer than necessary.
- **Resource lifetime:** Close response bodies and streams, release semaphores and locks on every return path, stop timers when appropriate, and respect context cancellation.
- **Bounds and growth:** Check integer conversions, size limits, `len+1` overflow, unbounded reads, recursion, and attacker-controlled allocations before allocating or reading.
- **Escape behavior:** For allocation-sensitive code, inspect compiler escape output for the affected package when it can clarify ownership or unexpected heap movement:

  ```sh
  go test -gcflags='-m=2' ./path/to/pkg 2> /tmp/protoreflect-escape.txt
  ```

- **Pointer checking:** When code uses `unsafe`, cgo, manual pointer conversion, or subtle aliasing, also run the focused suite with strict pointer checks:

  ```sh
  go test -gcflags=all=-d=checkptr=2 ./path/to/pkg
  ```

If a listed check is not applicable or cannot run in the environment, state that explicitly in the handoff; do not silently omit it.

## Required handoff evidence

Summarize:

- production path changed and why it matters;
- benchmark command, before/after `ns/op`, `B/op`, and `allocs/op`;
- compatibility and ownership invariants covered by tests;
- exact verification commands and their outcomes;
- any residual risk, noisy benchmark, skipped check, or accepted trade-off.

Do not describe work as faster, safer, or complete without this evidence.
