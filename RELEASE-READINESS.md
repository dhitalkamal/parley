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
- [ ] add CI: run go build, go vet, go test, go test -race, and deadcode on
      every push and PR. the project's own pitch is "runs in CI" but it has
      no CI - a broken commit can currently be tagged and shipped.
- [ ] add a LICENSE file. the repo is public and distributed via brew/curl/
      go install, but with no license it is all-rights-reserved by default,
      which blocks the homebrew-tap and installer model.
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
- [ ] history.jsonl is unbounded on disk and fully re-parsed on every history
      modal open (the "last 50" is display-only). add a disk cap or windowed
      read, matching the WS transcript cap pattern.

### test coverage gaps (foundational, thin)
- [ ] internal/platform/fsstore is at ~17 percent. it is the shared storage
      foundation every on-disk store sits on. raise it.
- [ ] internal/netcheck (~43 percent) and cmd/parley TUI-launch path (~45
      percent, partly inherent since the TUI needs a tty). cover what can be
      covered without a tty.

### error-path UX
- [ ] audit user-facing error messages for actionability. the pre-fix
      `could not open a new TTY` was cryptic; sweep for similar.
- [ ] responsive sidebar: below ~80 columns the fixed-width sidebar takes
      half the screen. add a breakpoint to hide or shrink it.

### secrets
- [ ] masking now covers common secret keys and the headers tab, but the
      Cookies tab still renders cookie values. decide whether to mask there
      too.
- [ ] history.jsonl and last_responses.json are now 0600 but still store
      credentials in cleartext on disk. decide whether at-rest values should
      be redacted, not just permission-gated.

## nice-to-have (post-1.0)

- [ ] responsive/adaptive layouts beyond the current graceful floor.
- [ ] configurable transcript/history caps.
- [ ] a `parley --help` that also documents PARLEY_HOME / XDG_CONFIG_HOME.
