/**
 * Vectos — OpenCode plugin adapter
 *
 * Thin layer that connects OpenCode's event system to the Vectos HTTP server.
 * Tracks file changes during a session and triggers incremental reindex
 * via HTTP POST to the local Vectos server — async, silent, no TUI pollution.
 *
 * Flow:
 *   OpenCode file events → accumulate changed paths → debounce timer →
 *   POST /reindex → SQLite updated (model stays in memory)
 *
 * The plugin auto-starts the Vectos server if it's not running.
 *
 * It also injects the "Prefer Vectos MCP" guidance into the system prompt of
 * every agent via the `experimental.chat.system.transform` hook. This is
 * universal: it works for the orchestrator and for sub-agents spawned by any
 * plugin (oh-my-opencode-slim, custom agents, or plain opencode), regardless
 * of the model. AGENTS.md only reaches the main agent; this hook guarantees
 * sub-agents also receive the Vectos-first instruction.
 *
 * Installed by: vectos setup opencode
 * Removed by:   vectos setup opencode --uninstall
 */

import type { Plugin } from "@opencode-ai/plugin"

// ─── Configuration ───────────────────────────────────────────────────────────

const VECTOS_PORT = parseInt(process.env.VECTOS_PORT ?? "7438")
const VECTOS_URL = `http://127.0.0.1:${VECTOS_PORT}`
const VECTOS_BIN = process.env.VECTOS_BIN ?? Bun.which("vectos") ?? "vectos"
const DEBOUNCE_MS = parseInt(process.env.VECTOS_DEBOUNCE_MS ?? "3000")
const MAX_CHANGED_PATHS = 200 // Safety limit per reindex batch
const HEALTH_CACHE_MS = 5000

// ─── Guidance ────────────────────────────────────────────────────────────────

/**
 * Vectos-first guidance injected into every agent's system prompt.
 *
 * Kept inline (not read from AGENTS.md) so sub-agents receive it even when
 * AGENTS.md is not propagated to them. The same text is managed in
 * internal/setup/guidance_content.go for AGENTS.md injection; keep these two
 * copies in sync when editing.
 */
const VECTOS_GUIDANCE = `## Prefer Vectos MCP

When Vectos MCP tools are available for a project, prefer the Vectos search tools before using \`grep\`, \`find\`, \`glob\`, or broad file reads.

For source code and implementation lookups, use \`vectos_search_code\` when your client prefixes MCP tools with the server name, or \`search_code\` when it does not.

For README files, API docs, ADRs, and other documentation, use \`vectos_search_docs\` or \`search_docs\` instead of mixing docs into broad file searches.

If the project is not yet indexed or results are not useful, run \`vectos_index_project\` or \`index_project\` and retry. When you need documentation retrieval, index docs separately with \`docs: true\`.

Use \`grep\`, \`glob\`, and direct file reads only as a fallback when Vectos has no useful results or when you need exact pattern matching.

If you create, move, or edit files while working, prefer an incremental refresh with \`changed\` paths before retrying search. Use a full reindex only when the affected scope is broad or uncertain.

When delegating to specialist agents that perform code search, explicitly instruct them to use Vectos search tools before \`grep\`/\`glob\`. Sub-agents may not automatically receive tool-preference instructions — the main agent is the enforcement point. If a sub-agent returns without using Vectos, remind it in the next delegation.

## Nx Monorepo — Lib Coverage

When working inside an Nx monorepo, Vectos includes all internal dependency libs in the resolved scope by default. Only projects with Nx type \`"e2e"\` are excluded. Set \`VECTOS_NX_INCLUDE_E2E=1\` to override this exclusion.

Use the \`project\` parameter to scope search and indexing to a specific Nx project:

- \`search_code\` and \`search_docs\`: pass \`project: "<project-name>"\` to scope results to that project and its internal dependencies.
- \`index_project\`: pass \`project: "<project-name>"\` to index a specific Nx project. Vectos automatically indexes the project and all its internal dependency libs.
- \`list_projects\`: call this tool to discover available Nx project names in the workspace before searching or indexing.

If the search returns guidance \`IDX_MISSING\`, index the project first with \`index_project\` using the correct \`project\` name.

If you edit a **shared lib**, its changes are reflected in every project that depends on it. If searches in a downstream project still feel stale after refreshing the lib's project index, refresh the downstream project index as well — or perform a full reindex if the blast radius is unclear.`

// ─── HTTP Client ─────────────────────────────────────────────────────────────

async function vectosFetch(
  path: string,
  opts: { method?: string; body?: any } = {}
): Promise<any> {
  try {
    const res = await fetch(`${VECTOS_URL}${path}`, {
      method: opts.method ?? "GET",
      headers: opts.body ? { "Content-Type": "application/json" } : undefined,
      body: opts.body ? JSON.stringify(opts.body) : undefined,
    })
    return await res.json()
  } catch {
    // Vectos server not running — silently fail
    return null
  }
}

async function isVectosRunning(): Promise<boolean> {
  try {
    const res = await fetch(`${VECTOS_URL}/health`, {
      signal: AbortSignal.timeout(500),
    })
    return res.ok
  } catch {
    return false
  }
}

async function waitForVectosReady(): Promise<boolean> {
  const deadline = Date.now() + 5000
  while (Date.now() < deadline) {
    if (await isVectosRunning()) return true
    await new Promise((r) => setTimeout(r, 250))
  }
  return false
}

// ─── Plugin Export ──────────────────────────────────────────────────────────

export const VectosPlugin: Plugin = async (ctx) => {
  // Accumulated changed file paths, flushed on session.idle or debounce
  const changedFiles = new Set<string>()
  let reindexTimer: ReturnType<typeof setTimeout> | null = null
  let lastHealthCheck = 0
  let vectosAvailable = false

  async function isVectosAvailable(): Promise<boolean> {
    const now = Date.now()
    if (now - lastHealthCheck < HEALTH_CACHE_MS) return vectosAvailable

    lastHealthCheck = now
    vectosAvailable = await isVectosRunning()
    return vectosAvailable
  }

  /**
   * Send accumulated changed paths to the Vectos server for reindexing.
   * Completely silent — no console output to avoid polluting the OpenCode TUI.
   *
   * Detects whether changed files are code or docs and sends separate
   * reindex requests for each type so the correct database is updated.
   */
  async function triggerReindex(): Promise<void> {
    if (changedFiles.size === 0) return

    const paths = [...changedFiles].slice(0, MAX_CHANGED_PATHS)
    changedFiles.clear()

    const changedArg = paths.join(",")

    // Reindex code database
    await vectosFetch("/reindex", {
      method: "POST",
      body: {
        path: ctx.directory,
        changed: changedArg,
      },
    })

    // Also reindex docs database (docs files may have changed too)
    await vectosFetch("/reindex", {
      method: "POST",
      body: {
        path: ctx.directory,
        changed: changedArg,
        docs: true,
      },
    })
  }

  /**
   * Schedule a debounced reindex. Resets the timer on each call so
   * rapid bursts of file changes get batched into a single reindex.
   */
  function scheduleReindex(): void {
    if (reindexTimer !== null) {
      clearTimeout(reindexTimer)
    }
    reindexTimer = setTimeout(() => {
      reindexTimer = null
      triggerReindex()
    }, DEBOUNCE_MS)
  }

  // Try to start Vectos server if not running
  const running = await isVectosAvailable()
  if (!running) {
    try {
      Bun.spawn([VECTOS_BIN, "serve"], {
        stdout: "ignore",
        stderr: "ignore",
        stdin: "ignore",
      })
      // Wait and retry health until the server is ready.
      vectosAvailable = await waitForVectosReady()
      lastHealthCheck = Date.now()
    } catch {
      // Binary not found or can't start — plugin will silently no-op
    }
  }

  return {
    // ─── System Prompt Injection ─────────────────────────────────────
    // Universal: runs for every agent (orchestrator + sub-agents) on
    // every LLM call. Only injects when Vectos is reachable; the health
    // check is cached briefly to avoid a localhost request per turn.
    "experimental.chat.system.transform": async (_input, output) => {
      if (await isVectosAvailable()) {
        output.system.push(VECTOS_GUIDANCE)
      }
    },

    // ─── File Change Events ──────────────────────────────────────────
    // Accumulate changed files silently. No reindex until debounce/idle.

    event: async ({ event }) => {
      const eventType = (event as any).type as string

      // file.edited: emitted when OpenCode edits a file (Write/Edit tools)
      if (eventType === "file.edited") {
        const filePath = (event as any).properties?.file as string | undefined
        if (filePath) {
          changedFiles.add(filePath)
          scheduleReindex()
        }
      }

      // file.watcher.updated: emitted by the filesystem watcher
      if (eventType === "file.watcher.updated") {
        const filePath = (event as any).properties?.file as string | undefined
        const fileEvent = (event as any).properties?.event as string | undefined

        if (filePath && (fileEvent === "add" || fileEvent === "change" || fileEvent === "unlink")) {
          changedFiles.add(filePath)
          scheduleReindex()
        }
      }

      // session.idle: flush any pending changes immediately
      if (eventType === "session.idle") {
        if (reindexTimer !== null) {
          clearTimeout(reindexTimer)
          reindexTimer = null
        }
        await triggerReindex()
      }
    },
  }
}
