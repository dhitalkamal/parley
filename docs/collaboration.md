# sharing collections with a team (git / GitHub)

parley stores collections and environments as plain files on disk, so a team
can share and co-edit them through an ordinary git repository - no server, no
account, no parley-specific service. This is async collaboration: you pull your
teammates' changes and push your own, the same as sharing code.

Real-time simultaneous editing (two cursors in one request at once) is out of
scope by design - parley is local-first with no backend.

## what is safe to share, and what is not

A parley home (the `ProjectRoot` of a workspace) holds a mix of shareable
content and local-only state:

Share (tracked in git):

- `collections/` - your requests and folders. This is the point of the repo.

Keep local (git-ignored - see the template below):

- `workspaces.json` - machine-specific: it stores absolute paths.
- `active_environment` - a per-user choice, not a team fact.
- `history.jsonl`, `last_responses.json`, `runs.jsonl` - per-user runtime logs.
  These can contain credentials and full response bodies, so they must never be
  committed.
- `environments/`, `globals.json` - environment values can hold secrets
  (tokens, passwords). They are kept local for now. Sharing environment
  structure safely is the secret-split model described under "roadmap" below,
  which is not built yet.

### secret hygiene for the files you do share

Request files under `collections/` are shared verbatim. If you type a secret
directly into a request (a hardcoded `Authorization` header, a password in a
body), that secret goes into the repo. Use `{{variables}}` for anything secret
and resolve them from a local environment instead, so the shared request
carries only the variable name. Review a diff before you push.

## the .gitignore

Copy `docs/gitignore.template` to `.gitignore` at the root of your shared repo
(the workspace `ProjectRoot`), or paste this:

```
# parley shared-collection ignore list.
# share: collections/ (requests and folders).
# keep local: per-user state, runtime logs, and anything holding secrets.

# per-user / machine-specific
workspaces.json
active_environment

# runtime logs and caches (can contain credentials and response bodies)
history.jsonl
last_responses.json
runs.jsonl

# environment values can hold secrets - kept local until the secret-split
# model lands (see docs/collaboration.md "roadmap").
environments/
globals.json
```

## workflow

Keep the shared repo separate from your default `~/.config/parley` by pointing
a dedicated workspace at its own directory.

### owner (first time)

1. In parley, create a workspace whose project directory is a fresh path, e.g.
   `~/src/team-api-collections` (Workspaces screen, `f4`).
2. Add some requests so `collections/` has content.
3. In that directory: `git init`, add the `.gitignore` above, then
   `git add collections .gitignore && git commit -m "initial collection"`.
4. Create an empty repo on GitHub and push:
   `git remote add origin git@github.com:org/team-api-collections.git` then
   `git push -u origin main`.

### teammate (joining)

1. `git clone git@github.com:org/team-api-collections.git ~/src/team-api-collections`
2. In parley, create a workspace pointing at that cloned directory.
3. Their requests appear in the sidebar. Set up their own environments locally
   (URLs and secrets), since those are not shared yet.

### day to day

- `git pull` before you start, so you have the latest requests. parley reads
  the files on the next refresh.
- `git add`, `git commit`, `git push` your changes as usual.
- Two people editing the same request produces a normal git merge conflict in
  that request's JSON file - resolve it like any other conflict. The manual
  order prefix in filenames (`010_name`) means reordering shows up as renames;
  keep reorders in their own small commits to reduce churn.

## roadmap

This document plus the ignore template is phase 1 (works today with plain git).
Planned follow-ups, in order:

- secret-split for environments: parley writes each environment as a shareable
  template (variable names and non-secret values) plus a git-ignored local
  overlay holding only the secret-flagged values. Teammates get the structure,
  never the secrets. This changes the on-disk environment format, so it is
  gated on the "freeze or version the storage format" decision.
- in-app sync: a TUI command (and `parley` subcommand) that runs status / pull
  / commit / push against the workspace's git repo by shelling out to the
  user's own git, so credentials and SSH keys are reused rather than
  reimplemented. Surfaces conflicts and reloads the tree after a pull.
