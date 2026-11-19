# Workspaces for parallel agents: worktrees, copy-on-write checkouts and shared services

Research date: 2026-11-18. The question: how should several coding agents, run at once on this
Mac, each get a checkout of nitrovin that does not collide with the others, starts warm and is
cleaned up afterwards? Three surveys answered it: git worktrees and the agent tools built on
them, filesystem-level isolation (ovnfs, APFS clones, jj, VMs), and what in nitrovin and the
local tooling breaks when two checkouts run at once.

**Decision (2026-11-18): worktrunk.** Session worktrees use [worktrunk](https://github.com/max-sixty/worktrunk)
(`wt`, Homebrew). Its user config puts worktrees at `~/Projects/<org>/.worktrees/<repo>/<branch | sanitize>`.
The `wt` scripts are gone ([dotfiles#142](https://github.com/joshuasprow/dotfiles/pull/142)). Flattening
the branch's `/` puts each worktree one level deep, where `omp worktree list` classes it as a
worktree rather than `stray`. `wt step prune --dry-run` already detects the drift listed below,
squash-merged branches included. ovnfs stays under review until
[#79](https://github.com/stencil-hq/ovnfs/issues/79) is fixed.

## Summary

- **Every first-party agent tool that edits locally isolates with plain `git worktree`.** Claude
  Code, Codex, Cursor, Conductor, vibe-kanban and ccmanager all do. Each tool keeps its own root
  and cleanup policy.
- **The tools have converged on two conventions:**
  - `.worktreeinclude`, which copies a file only if it is both matched and ignored by git.
  - A setup command run once per workspace, with an optional teardown.

  None of them isolates ports, containers or databases. They hand over a hook and a variable
  (`CONDUCTOR_PORT`, worktrunk's `hash_port`).
- **At nitrovin's size, copy-on-write saves a cost that does not exist here.** The repo is 352
  tracked files (about 11 MB, `.git` 27 MB). Measured here, `cp -c -R` took 0.09 s and
  `git clone --shared` took 0.06 s. The Go build and module caches are global and safe for
  concurrent use.
- **What does cost:**
  - Shared services with fixed names and ports, and tests that mutate a shared database.
  - The 630 MB of `deploy/cloudflare` dependencies in a worktree that ran Worker work.
  - Drift: worktrees that are detached, prunable, or named after a different branch.
- **ovnfs** is the most interesting new option, and omp already integrates it. It is 13 days
  old, and it has an open bug where a dead daemon hard-hangs every process that touches the
  mount. Re-evaluate it once that is fixed.

## Two layers of workspace

The work splits into two layers, and each has its own tool:

| Layer | Lifetime | Who creates it | Fit |
| --- | --- | --- | --- |
| **Session worktree**: one per issue or branch, with a Tern tab and a human reviewing the PR | days | a person or a lead agent | worktrunk (`wt switch -c`) under `~/Projects/joshuasprow/.worktrees/nitrovin/<branch>` |
| **Subagent checkout**: one per `task` subagent inside a session | minutes | omp | omp `task.isolation` (APFS clone today; ovnfs when `auto` finds it) |

## Evidence

### git worktree

- **One branch per worktree.** `add` refuses a branch that is checked out elsewhere. `-d` makes a
  detached throwaway.
- **Lifecycle:**
  - `remove` refuses a dirty tree.
  - `prune` clears the metadata of directories deleted by hand.
  - `lock` (or race-free `add --lock`) blocks prune, move and remove.
  - `list --porcelain -z` is the script-stable format.
  - ([git-worktree](https://git-scm.com/docs/git-worktree))
- **Shared state:** config, refs, `info/exclude` and hooks are shared. Per-worktree config needs
  `extensions.worktreeConfig`, which older gits refuse.
  ([gitrepository-layout](https://github.com/git/git/blob/master/Documentation/gitrepository-layout.adoc))
- **Paths:** `worktree.useRelativePaths` (git 2.48) only matters if worktrees are moved or mounted
  into containers.
  ([RelNotes 2.48](https://github.com/git/git/blob/v2.48.0/Documentation/RelNotes/2.48.0.txt))
- **Hooks:** `post-checkout` also runs after `worktree add`, but it fires on every ordinary
  checkout too. ([githooks](https://github.com/git/git/blob/master/Documentation/githooks.adoc))
- **Not copied:** gitignored files. nitrovin has no submodules.
- **Go:** `./...` skips directories starting with `.`
  ([helpdoc.go](https://github.com/golang/go/blob/master/src/cmd/go/internal/help/helpdoc.go)).

### Agent tools and worktree managers

| Tool | Root, HEAD | Setup, cleanup |
| --- | --- | --- |
| [Claude Code](https://code.claude.com/docs/en/worktrees) | `.claude/worktrees/<name>`, branch `worktree-<name>` | `.worktreeinclude`. `WorktreeCreate` hook replaces the git logic. Locks while running. Sweeps old subagent worktrees, never hand-made ones. |
| [Codex app](https://learn.chatgpt.com/docs/environments/git-worktrees) | `$CODEX_HOME/worktrees`, detached | `.worktreeinclude`, per-OS setup scripts. Keeps the 15 most recent. |
| [Cursor](https://cursor.com/docs/configuration/worktrees) | its own root | `.cursor/worktrees.json`. Caps at 25 per machine and **deletes worktrees in its root that it did not create**. |
| [Conductor](https://www.conductor.build/docs/reference/scripts) | `~/conductor/workspaces/<repo>/<ws>` | Setup and archive scripts. `CONDUCTOR_PORT` gives 10 ports per workspace. "Nonconcurrent" mode for one shared DB. |
| [worktrunk](https://github.com/max-sixty/worktrunk) | path template | Hooks: `pre-start`, `post-start`, `pre-remove`, `pre-merge`. `wt step copy-ignored` reflinks `.worktreeinclude` files. `hash_port`. `wt merge`, `wt step prune`. Has an oh-my-pi plugin. 12.6k★, Apache-2.0, v0.82.0 (2026-11-14), pre-1.0. |
| [gwq](https://github.com/d-kuro/gwq), [ccmanager](https://github.com/kbwo/ccmanager), [vibe-kanban](https://github.com/BloopAI/vibe-kanban) | worktrees | Copy lists and setup commands. vibe-kanban garbage-collects worktrees for finished tasks. |
| [container-use](https://github.com/dagger/container-use) | branch + Dagger container | "Early development"; last release 2025-08. |

`.worktreeinclude` means the same thing in Claude Code, Codex, Conductor, ccmanager and
worktrunk: copy a file only if it matches a pattern and git ignores it. Preview the selection:

```sh
git ls-files --others --ignored --exclude-from=.worktreeinclude
```

On APFS, `cp -c` clones copy-on-write: worktrunk measured 14 GB in 20 s with no extra disk,
against 2 min with plain `cp -R` ([step](https://worktrunk.dev/step/)). Cursor advises against
symlinked dependencies.

### Filesystem-level isolation

- **[ovnfs](https://github.com/stencil-hq/ovnfs)** (stencil-hq, the org behind omp and Tern).
  v0.8.0, created 2026-11-05, 268★, MIT, about 75% of commits from one author.
  - **How it works:**
    - One user-space daemon serves copy-on-write checkouts. On macOS it uses an NFSv3 loopback
      over a Unix socket, so it needs no kext and no root.
    - Checkouts appear under `~/checkouts/<name>`, configured in `.o2/ovnfs.yml`.
    - Ignored and cache directories come from a background APFS clone snapshot.
    - Each checkout gets a private git directory whose objects come through alternates.
    - Work comes back via `apply` (three-way), `land` (commits) or `git fetch ovnfs/NAME`.
    - Old checkouts are cleaned with `prune --idle 7d`.
    - ([README](https://github.com/stencil-hq/ovnfs#readme), [checkouts.md](https://github.com/stencil-hq/ovnfs/blob/main/docs/checkouts.md))
  - **By default it changes the source repo:** it writes `.git/ovnfs-remotes` and an include line
    into `.git/config`.
  - **It is "not a sandbox".** Absolute paths reach the host. NFSv3 cannot push invalidations, so
    edits behind the mount can stay invisible for up to 60 s
    ([limitations.md](https://github.com/stencil-hq/ovnfs/blob/main/docs/limitations.md)).
  - **Open issues on 2026-11-18:**
    - [#79](https://github.com/stencil-hq/ovnfs/issues/79): a dead daemon leaves a hard mount, and
      "every process touching it blocks forever".
    - [#82](https://github.com/stencil-hq/ovnfs/issues/82): fsynced files are lost after the
      daemon is SIGKILLed.
  - **Not installed** on this Mac.
- **omp integrates ovnfs** ([omp://tools/task.md](omp://tools/task.md), "Isolation"). Once
  `ovnfs` is on PATH:
  - omp's system prompt tells *every* agent to use `ovnfs new/diff/apply/rm` instead of
    `git worktree add`, `git clone` or `cp -r`.
  - `isolation.backend=auto` puts isolated subagents in ovnfs checkouts.

  Installing the binary is therefore a global behaviour change, not an opt-in per repo. Without
  ovnfs, `auto` falls back to `apfs`, `reflink` or `rcopy`. Two settings decide how the work comes
  back:
  - `task.isolation.merge` returns it as patches or as an `omp/task/<id>` branch.
  - `task.isolation.apply` decides whether it is applied.
- **APFS clonefile and `git clone --shared`** are instant at this size. A `cp -c` clone has its
  own `.git`, so it needs a fetch to share branches. Alternates corrupt the clone if the source
  prunes objects it borrows ([git-clone](https://git-scm.com/docs/git-clone)).
- **Other options and why they don't fit:**
  - [jj workspaces](https://github.com/jj-vcs/jj/blob/main/docs/working-copy.md) mean adopting jj.
  - Sapling's EdenFS is not publicly supported.
  - overlayfs, btrfs and ZFS are Linux-only.
  - AgentFS (beta) and LayerFS (no crash durability) are young.
- **VMs** ([Docker Sandboxes](https://docs.docker.com/ai/sandboxes/get-started),
  [Apple container](https://github.com/apple/container)) give each agent its own Docker daemon
  and network. The cost: cold Go caches unless the host caches are mounted in, and gopls and the
  editor outside the agent.

### Go caches

- `go help cache`: "safe for concurrent invocations".
- Without `-trimpath`, go1.27.4's `buildActionID` hashes the package directory
  (`cmd/go/internal/work/exec.go`). A new workspace path therefore recompiles nitrovin's own
  packages once, while dependencies stay warm. [INFERENCE] That costs seconds.

### nitrovin under concurrent checkouts

| Resource | Where | Concurrent? |
| --- | --- | --- |
| Oracle container `nhtsa-oracle` | `scripts/oracle.sh:76` (`ORACLE_CONTAINER` overrides) | Decodes are reads, so one shared container works. `up`, `down`, `restore` and `freshen` mutate it. |
| Stock Postgres `nitrovin-stock`, port 55432 | `CONTRIBUTING.md:393` | Collides. The tests `DROP SCHEMA`/`DROP ROLE` (`pkg/stock/pg/postgres_test.go:89`), so each workspace needs its own database. |
| `nitrovin serve` on 8080 | `cmd/nitrovin/serve.go:80` (`$PORT`, `--addr`) | Collides only by hand; tests use port 0. |
| `wrangler dev` on 8787; images `nitrovin-dev:vcs`/`:devel` | `deploy/cloudflare/worker/dev-test.sh:24,84` | Port and tags collide. It also rewrites the tracked worker `Dockerfile` in place until exit. |
| bench MSSQL `nitrovin-bench-mssql` | `scripts/bench.sh:49` (`MSSQL_CONTAINER`) | Collides. |
| chromedp profiles, test binaries, bench files | `os.MkdirTemp` | Safe. |
| CI | concurrency keyed on `workflow-ref`; one global `deploy-nitrovin-app` | Safe with one branch per workspace. |

**A fresh worktree lacks:**

- `cmd/nitrovin/sw/`: `scripts/sw.sh`, plus `scripts/epa.sh` (network), are needed only to serve
  the page with visitor files.
- The Worker's `node_modules`: needs `npm ci`, and only for Worker work.
- `AGENTS.md`, which is untracked and not on `origin/main`.

**Already handled:**

- mise trust carries over to worktrees ("trusting a config file in a repository's main checkout
  also trusts the same path in its git worktrees", `mise trust --help`, 2026.11.6).
- direnv binds `OMP_PROFILE` and `OMP_WORKTREE_DIR` for any path under `~/Projects/joshuasprow/`.
- nitrovin has no `.env` or `.envrc`.
- `/tmp` worktrees get none of these.

### Local conventions and drift (as found, before worktrunk)

- **`wt <branch> [base]`** (dotfiles, since removed) created `~/Projects/<org>/.worktrees/<repo>/<branch>`,
  and `wt --tab` opened a Tern tab.
  - `wt-prune` is a dry run by default. It removes clean worktrees whose PR merged or whose
    upstream is gone, and never deletes branches.
  - `wt-remove` refuses the main clone and dirty trees.
- **`OMP_WORKTREE_DIR`** points omp at the same root.
- **Hazard:** `omp worktree clear --dry-run` would remove `.worktrees/nitrovin` and
  `.worktrees/nitrovin-data` entirely. omp labels them `stray` because `task/<name>` branches nest
  one directory deeper than it expects (skill://omp-cli warns about this).
- **Drift on 2026-11-18** (cleared the same day by `wt step prune`, `rmdir` and a switch to `main`):
  - The main checkout is on `task/column-check-310` (PR merged), with `AGENTS.md` untracked, so
    every stamped build there is `+dirty`.
  - `/private/tmp/nv-main-src` is prunable.
  - `task/cloudflare-104` is detached at a commit already on main.
  - Directory `task/do-latency-126` holds branch `task/smoke-rollout-127`.
  - 9 local `task/*` branches are merged or closed but not deleted.
    `git branch --merged` cannot see them, because PRs squash-merge.
  - Empty category directories (`build/`, `fix/`, `research/`, …) remain under
    `.worktrees/nitrovin/`.

## Options

1. **`wt` plus a committed setup script.** Keep the existing root and naming, and add
   `scripts/worktree.sh` for setup, teardown and per-workspace service names. It needs no new
   dependency and keeps one convention. You maintain the lifecycle yourself, and `wt-prune` is
   already most of it.
2. **worktrunk.** Point its `worktree-path` template at the same root and commit `.config/wt.toml`
   hooks and `.worktreeinclude`. You get `hash_port`, `wt merge` and an omp plugin. Against that:
   it is pre-1.0, its hooks need approval, and it would sit beside `wt` as a second convention
   unless `wt` is retired.
3. **ovnfs for everything.** Instant checkouts, `apply`/`land`/`prune`, and agents never touch the
   main checkout. Against that: an NFS daemon two weeks old in every editor's I/O path (#79), it
   mutates `.git/config`, and installing it changes every omp session's instructions.
4. **Each agent tool's own worktrees** (`.claude/worktrees`, `$CODEX_HOME`, Cursor). Three roots
   and three cleanup policies, and Cursor deletes worktrees it did not create.
5. **VM per agent.** Full isolation of Docker and the network, but the Go caches start cold and
   the editor sits outside the VM. That cost is worth paying only if agents need to run the
   oracle concurrently.

None of the options isolates services by itself. Whichever is chosen, per-workspace service
names come from the workspace:

- One read-only `nhtsa-oracle` serves everyone; run `restore`/`freshen` from one place at a time.
- Stock Postgres gets a container and port per workspace, or one container with a database per
  workspace.
- `PORT`, `ORACLE_CONTAINER` and `MSSQL_CONTAINER` are set per workspace.
- `dev-test.sh` runs one at a time.
