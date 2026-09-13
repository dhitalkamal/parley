# parley 1.0 readiness

Status of parley toward a 1.0 release. Produced from a QA program in
2026-09: an automated multi-angle review, a live TUI QA pass, and an
adversarially-verified fix round. Living doc - update as items close.

## done

- full codebase review: architecture map + bug/security/perf/maintainability
  review, adversarially verified, plus a gap critic. 22 findings.
- live TUI QA (tmux) across terminal sizes down to 20x8: no crashes, layout
  degrades gracefully. render, help overlay, palette, tree nav, tab switch,
  body editor, dashboard all work.
- 22 findings fixed and consolidated (PR: qa-remediation), each with a test.
- dead code eliminated: deadcode ./... reports zero unreachable functions.
- verified: go build, go vet, go test, go test -race all pass.
- CLI fixes shipped separately: no-path `parley run` runs the whole
  collection, `parley version`/`help`, version stamping via ldflags.

## must-do before 1.0

### release safety
- [x] add CI: go build, go vet, go test -race, and a deadcode gate run on
      every push to main and every PR (.github/workflows/ci.yml). verified
      green on GitHub before merge.
- [x] add a LICENSE file. MIT, added at repo root with a README section.
- [ ] confirm the release build injects the version ldflag (the committed
      .goreleaser.yaml does this; run `goreleaser check` on the release host
      and cut a tagged release so shipped binaries stop reporting "dev").

### storage format
- [ ] decide and document whether the on-disk format (collections/,
      environments/, workspaces.json, jsonl histories) is frozen for 1.0.
      README says it "may still change" - 1.0 should commit to it or version
      it.
- [ ] migration story for the file-permission tightening (0644 -> 0600):
      existing users' files keep their old permissions until rewritten.
      decide whether to proactively re-chmod on startup.

### perf follow-ups (partials that were improved but not fully closed)
- [ ] websocket transcript render is bounded (cap 5000 + block cache) but
      each refresh is still O(n) string-building. windowing to the visible
      viewport would make it O(visible). acceptable for 1.0, revisit if a
      high-rate socket lags.
- [ ] dashboard overview is now cached; the run-history filter/scan and the
      performance section still scan on filter changes. fine at current
      scale, revisit if run history grows large.
- [x] history.jsonl is now capped on disk (maxHistoryEntries, oldest evicted
      on append via an atomic rewrite), which also bounds ListHistory's parse
      cost. append is per-send, not a hot path, so trimming there is cheap.

### test coverage gaps (foundational, thin)
- [x] internal/platform/fsstore raised from ~17 to 100 percent - round-trip
      tests for RequestToFile/RequestFromFile (HTTP, WebSocket, zero-value,
      disabled auth/refresh gating, timeout conversion).
- [x] internal/netcheck raised from ~43 to ~91 percent (defaultListInterfaces
      and hasRoutableAddr covered against the real loopback interface). the
      remaining gap is the routable-IP-present branch, which needs a real
      routed interface the test host may not have.
- [ ] cmd/parley TUI-launch path (~45 percent) is partly inherent since the
      TUI needs a tty; cover what can be covered without one.

### error-path UX
- [ ] audit user-facing error messages for actionability. the pre-fix
      `could not open a new TTY` was cryptic; sweep for similar.
- [ ] responsive sidebar (DECISION): a breakpoint already exists - the
      sidebar drops out entirely below minWidthForSidebar (60 cols). the open
      question is the 60-90 col band, where the fixed-width sidebar is present
      but proportionally large. shrinking it or raising the threshold there is
      a UX-preference call (a higher threshold hides it for normal terminals),
      so it is left for a deliberate decision rather than guessed.

### secrets
- [x] the Cookies tab now masks cookie values by default (session tokens are
      inherently sensitive), revealable with the same ctrl+u toggle. Path/
      Domain/flags stay visible.
- [ ] history.jsonl and last_responses.json are now 0600 but still store
      credentials in cleartext on disk. decide whether at-rest values should
      be redacted, not just permission-gated.

## nice-to-have (post-1.0)

- [ ] responsive/adaptive layouts beyond the current graceful floor.
- [ ] configurable transcript/history caps.
- [ ] a `parley --help` that also documents PARLEY_HOME / XDG_CONFIG_HOME.
