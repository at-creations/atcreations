# AT Creations monorepo

Storefront (catalogue) + admin dashboard + API for AT Creations. Read `docs/ARCHITECTURE.md` and `docs/SECURITY.md` before changing behaviour.

## Layout

| Path | What | Stack |
|---|---|---|
| `apps/api` | HTTP API (`cmd/api`, `openapi.yaml` is the contract) and the image worker (`cmd/imgworker`, gRPC) | Go 1.27, chi, mongo-driver v2, go-redis v9, clerk-sdk-go v2, R2 via S3 API, gRPC |
| `apps/storefront` | Public catalogue (SSR, en/vi) | SvelteKit 2, Svelte 5 runes, Tailwind 4, Paraglide |
| `apps/admin` | Staff dashboard | SvelteKit 2, Svelte 5, svelte-clerk, TanStack Query v6 |
| `packages/api-types` | TS types generated from `openapi.yaml` | openapi-typescript |
| `deploy/` | Self-hosted production stack | Docker Compose, Caddy, MongoDB 8 replica set, Redis 8 |

## Commands

```bash
pnpm install                     # JS deps (pnpm workspace)
pnpm dev:stack                   # whole dev stack in Docker with live reload (S3 = local RustFS)
pnpm dev:stack:tunnel            # same + Cloudflare Tunnel on dev*.atcreations.ca (setup: docker/dev/cloudflared/setup.sh)
docker compose -f docker-compose.dev.yml up -d mongo redis   # only the databases
cd apps/api && go run ./cmd/api  # API on :8080 (needs apps/api/.env)
pnpm dev:storefront / pnpm dev:admin
pnpm gen:api                     # after ANY change to apps/api/openapi.yaml
cd apps/api && go vet ./... && go test ./...
pnpm -r check && pnpm -r lint && pnpm -r test && pnpm -r build
```

## Project rules

- **Contract first:** change `apps/api/openapi.yaml`, run `pnpm gen:api`, then the Go handlers and the apps. CI fails on drift.
- **Never trust the client for money, roles or status.** Prices/totals are computed server-side; roles come only from Clerk `publicMetadata.role`.
- **Every admin mutation** goes through `auth.Require(<permission>)` and records `audit.Record(...)`.
- **Decode JSON with `httpx.DecodeJSON`** (rejects unknown fields → no mass assignment). Escape user text in regexes with `mongox.ContainsPattern`. Never build Mongo filters from raw maps of request data.
- **Catalogue/store writes must invalidate the Redis cache** (`s.cache.Invalidate(ctx, redisx.NSCatalog)` / `NSStore`). Redis holds only ephemeral data and every use must fail open.
- **Admin data fetching goes through TanStack Query** (`apps/admin/src/lib/queries`): use the query-key factory and invalidate/set queries after mutations.
- **Image uploads go through `internal/uploads`** (staging + durable job queue) and the gRPC image worker (`cmd/imgworker`, `internal/imageworker`), which decodes with limits, applies EXIF orientation and re-encodes to WebP via `media.Process`. Object keys are random or job-derived; never use client filenames. Every path must remove the original (worker on success/invalid, API on cancel/give-up, `Sweep` for leftovers).
- **gRPC contract first:** edit `apps/api/proto/`, then `go run github.com/bufbuild/buf/cmd/buf@v1.73.0 generate` in `apps/api` (commit `internal/gen`).
- Svelte: runes only; never `{@html}` with API data; run the Svelte autofixer on changed components.
- Secrets live in `.env` / `deploy/env/*.env` / `deploy/secrets/` / `docker/dev/cloudflared/` (cert.pem, credentials.json) — never read, print or commit them.
- The `clerk-backend-api` skill executes live API calls (including deletes). Only use it with a development (`sk_test_`) key and confirm before any mutation.
- Commit messages: Conventional Commits (`feat(api): ...`).

---

# Ruflo agent harness

## Rules

- Do what has been asked; nothing more, nothing less
- NEVER create files unless absolutely necessary — prefer editing existing files
- NEVER create documentation files unless explicitly requested
- NEVER save working files or tests to root — keep code inside `apps/*` or `packages/*`, docs in `docs/`
- ALWAYS read a file before editing it
- NEVER commit secrets, credentials, or .env files
- Keep files under 500 lines
- Validate input at system boundaries

## Ruflo Capability Brain & Implementation Loop

Ruflo is the coordination ledger and policy decision point. Claude Code is the
executor: after a Ruflo coordination call, continue implementing the task.

When it is registered, call
`guidance_brain({ mode: "recommend", task: "..." })` before complex Ruflo
work. Use its live registry instead of guessing tool names. Treat
`registered`, `configured`, `reachable`, `healthy`, and `authorized`
as separate facts. If the brain is unavailable, continue with the compatible
`guidance_recommend` tool, CLI discovery, and repository instructions.

Follow the returned loop:

1. Recall memory and ADR constraints.
2. Inspect source, runtime, dependencies, policy, and health.
3. Route to the smallest capable topology, agents, skills, and tools.
4. Plan acceptance criteria, safety envelope, ownership, and validation.
5. Execute in isolated scopes; the coding agent performs the work.
6. Test focused, regression, and failure paths.
7. Validate types, security, policy, compatibility, and artifacts.
8. Benchmark a source-bound candidate against a source-bound baseline.
9. Optimize measured bottlenecks without weakening safety.
10. Bind claims and evidence to exact source/build receipts.
11. Reconcile concurrent handoffs and disclose limitations.
12. Publish only through a separately authorized release gate.

### Concurrency and authority

- Never allow two writers in one worktree; give each writing agent an isolated
  worktree and explicit file ownership.
- Read-only research may run concurrently and report findings to the owner.
- Only the integration owner edits shared manifests and lockfiles or reconciles
  overlapping changes.
- A child may drop capabilities but cannot add tools, network, secrets, spend,
  concurrency, namespaces, or delegation depth.
- A lease or claim coordinates ownership; it does not authorize a side effect.
- Darwin, Flywheel, MetaHarness, memory, and neural systems may propose or
  evaluate candidates but cannot self-promote or expand their SafetyEnvelope.
- Bind tests, benchmarks, policy decisions, and release evidence to an exact
  commit or immutable dirty-worktree snapshot.

## Agent Comms (SendMessage-First Coordination)

Named agents coordinate via `SendMessage`, not polling or shared state.

```
Lead (you) ←→ architect ←→ developer ←→ tester ←→ reviewer
              (named agents message each other directly)
```

### Spawning a Coordinated Team

```javascript
// ALL agents in ONE message, each knows WHO to message next
Agent({ prompt: "Research the codebase. SendMessage findings to 'architect'.",
  subagent_type: "researcher", name: "researcher", run_in_background: true })
Agent({ prompt: "Wait for 'researcher'. Design solution. SendMessage to 'coder'.",
  subagent_type: "system-architect", name: "architect", run_in_background: true })
Agent({ prompt: "Wait for 'architect'. Implement it. SendMessage to 'tester'.",
  subagent_type: "coder", name: "coder", run_in_background: true })
Agent({ prompt: "Wait for 'coder'. Write tests. SendMessage results to 'reviewer'.",
  subagent_type: "tester", name: "tester", run_in_background: true })
Agent({ prompt: "Wait for 'tester'. Review code quality and security.",
  subagent_type: "reviewer", name: "reviewer", run_in_background: true })

// Kick off the pipeline
SendMessage({ to: "researcher", summary: "Start", message: "[task context]" })
```

### Patterns

| Pattern | Flow | Use When |
|---------|------|----------|
| **Pipeline** | A → B → C → D | Sequential dependencies (feature dev) |
| **Fan-out** | Lead → A, B, C → Lead | Independent parallel work (research) |
| **Supervisor** | Lead ↔ workers | Ongoing coordination (complex refactor) |

### Rules

- ALWAYS name agents — `name: "role"` makes them addressable
- ALWAYS include comms instructions in prompts — who to message, what to send
- Spawn ALL agents in ONE message with `run_in_background: true`
- After spawning, continue independent local work; wait only when a dependency
  genuinely blocks progress
- Do not poll repeatedly — agents message back or complete automatically
- Give every writing agent an isolated worktree and a non-overlapping file scope

## Swarm & Routing

### Config
- **Topology**: hierarchical-mesh (anti-drift)
- **Max Agents**: 15
- **Memory**: hybrid
- **HNSW**: Enabled
- **Neural**: Enabled

```bash
npx ruflo@3.59.12 swarm init --topology hierarchical --max-agents 8 --strategy specialized
```

### Agent Routing

| Task | Agents | Topology |
|------|--------|----------|
| Bug Fix | researcher, coder, tester | hierarchical |
| Feature | architect, coder, tester, reviewer | hierarchical |
| Refactor | architect, coder, reviewer | hierarchical |
| Performance | perf-engineer, coder | hierarchical |
| Security | security-architect, auditor | hierarchical |

### When to Swarm
- **YES**: 3+ files, new features, cross-module refactoring, API changes, security, performance
- **NO**: single file edits, 1-2 line fixes, docs updates, config changes, questions

### 3-Tier Model Routing

| Tier | Handler | Use Cases |
|------|---------|-----------|
| 1 | Agent Booster (WASM) | Simple transforms — skip LLM, use Edit directly |
| 2 | Haiku | Simple tasks, low complexity |
| 3 | Sonnet/Opus | Architecture, security, complex reasoning |

## Memory & Learning

### Before Any Task
```bash
npx ruflo@3.59.12 memory search --query "[task keywords]" --namespace patterns
npx ruflo@3.59.12 hooks route --task "[task description]"
```

### After Success
```bash
npx ruflo@3.59.12 memory store --namespace patterns --key "[name]" --value "[what worked]"
npx ruflo@3.59.12 hooks post-task --task-id "[id]" --success true --store-results true
```

### MCP Tools (use `ToolSearch("keyword")` to discover)

| Category | Key Tools |
|----------|-----------|
| **Memory** | `memory_store`, `memory_search`, `memory_search_unified` |
| **Bridge** | `memory_import_claude`, `memory_bridge_status` |
| **Swarm** | `swarm_init`, `swarm_status`, `swarm_health` |
| **Agents** | `agent_spawn`, `agent_list`, `agent_status` |
| **Hooks** | `hooks_route`, `hooks_post-task`, `hooks_worker-dispatch` |
| **Security** | `aidefence_scan`, `aidefence_is_safe`, `aidefence_has_pii` |
| **Hive-Mind** | `hive-mind_init`, `hive-mind_consensus`, `hive-mind_spawn` |

### Background Workers

| Worker | When |
|--------|------|
| `audit` | After security changes |
| `optimize` | After performance work |
| `testgaps` | After adding features |
| `map` | Every 5+ file changes |
| `document` | After API changes |

```bash
npx ruflo@3.59.12 hooks worker dispatch --trigger audit
```

## Agents

**Core**: `coder`, `reviewer`, `tester`, `planner`, `researcher`
**Architecture**: `system-architect`, `backend-dev`, `mobile-dev`
**Security**: `security-architect`, `security-auditor`
**Performance**: `performance-engineer`, `perf-analyzer`
**Coordination**: `hierarchical-coordinator`, `mesh-coordinator`, `adaptive-coordinator`
**GitHub**: `pr-manager`, `code-review-swarm`, `issue-tracker`, `release-manager`

Any string works as a custom agent type.

## Build & Test

- ALWAYS run tests after code changes
- ALWAYS verify build succeeds before committing

See *Commands* above (Go tests + `pnpm -r check lint test build`).

## CLI Quick Reference

```bash
npx ruflo@3.59.12 init --wizard           # Setup
npx ruflo@3.59.12 swarm init --v3-mode     # Start swarm
npx ruflo@3.59.12 memory search --query "" # Vector search
npx ruflo@3.59.12 hooks route --task ""    # Route to agent
npx ruflo@3.59.12 doctor --fix             # Diagnostics
npx ruflo@3.59.12 security scan            # Security scan
npx ruflo@3.59.12 performance benchmark    # Benchmarks
```

26 commands, 140+ subcommands. Use `--help` on any command for details.

## Setup

```bash
claude mcp add claude-flow -- npx -y ruflo@3.59.12 mcp start
npx ruflo@3.59.12 doctor --fix
```

> The background `daemon` is optional. It runs interval workers that each spawn
> a headless `claude` session, so it consumes tokens continuously. Start it only
> if you want those sweeps: `npx ruflo@3.59.12 daemon start` (self-stops after 12h
> by default; `--ttl 0` to disable, `daemon status --all` to audit running daemons).

**Agent tool** handles execution (agents, files, code, git). **MCP tools** handle coordination (swarm, memory, hooks). **CLI** is the same via Bash.
