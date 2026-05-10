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
 * Note: Vectos guidance is injected via AGENTS.md by `vectos setup opencode`,
 * not via this plugin's system prompt hook, to avoid duplication.
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
  const running = await isVectosRunning()
  if (!running) {
    try {
      Bun.spawn([VECTOS_BIN, "serve"], {
        stdout: "ignore",
        stderr: "ignore",
        stdin: "ignore",
      })
      // Wait and retry health until the server is ready.
      await waitForVectosReady()
    } catch {
      // Binary not found or can't start — plugin will silently no-op
    }
  }

  return {
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
