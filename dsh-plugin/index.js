/**
 * Host half of the qg workspace bridge.
 *
 * The qg CLI cannot drive the Electron window directly. This plugin exposes a
 * loopback-only HTTP bridge (token in ~/.qg/dsh-bridge.json) that creates or
 * finds a Workspace, then marks its title so the Client half can open it.
 *
 * The marker contains the request id and the workspace's original title so the
 * Client can restore a user-renamed workspace after opening it.
 */
import { createServer } from 'node:http'
import { chmod, mkdir, readFile, realpath, rename, stat, unlink, writeFile } from 'node:fs/promises'
import { randomBytes, randomUUID } from 'node:crypto'
import { homedir } from 'node:os'
import { basename, dirname, isAbsolute, join, resolve } from 'node:path'

export const name = 'qg-dsh-open'
export const inject = ['workspaceRegistry']

const BRIDGE_VERSION = 1
const MARKER_PREFIX = 'qg-open:'
const REQUEST_LIMIT = 64 * 1024
const MARKER_RE = /^qg-open:([^:]+):(.*)$/u

function bridgeFilePath() {
  return process.env.QG_DSH_BRIDGE ?? join(homedir(), '.qg', 'dsh-bridge.json')
}

function markerTitle(requestId, title) {
  return `${MARKER_PREFIX}${requestId}:${encodeURIComponent(title)}`
}

function cleanTitle(title) {
  const match = MARKER_RE.exec(title)
  if (match === null) return title
  try {
    return decodeURIComponent(match[2])
  } catch {
    return title
  }
}

function send(res, status, value) {
  const body = JSON.stringify(value)
  res.writeHead(status, {
    'content-type': 'application/json; charset=utf-8',
    'content-length': Buffer.byteLength(body),
    'cache-control': 'no-store',
  })
  res.end(body)
}

function readBody(req) {
  return new Promise((resolveBody, reject) => {
    const chunks = []
    let size = 0
    req.on('data', (chunk) => {
      size += chunk.length
      if (size > REQUEST_LIMIT) {
        reject(new Error('request body too large'))
        req.destroy()
        return
      }
      chunks.push(chunk)
    })
    req.on('end', () => resolveBody(Buffer.concat(chunks).toString('utf8')))
    req.on('error', reject)
  })
}

/**
 * Start the bridge when the Desktop host activates this plugin.
 * @param {import('@deepseek-ai/cordis').Context} ctx - Host context.
 */
export function apply(ctx) {
  const token = randomBytes(32).toString('base64url')
  const bridgePath = bridgeFilePath()
  /** @type {Map<string, { requestId: string, previousTitle: string }>} */
  const pending = new Map()
  let server
  let stopped = false

  const removeBridgeFile = async () => {
    try {
      const current = JSON.parse(await readFile(bridgePath, 'utf8'))
      if (current.token === token) await unlink(bridgePath)
    } catch {
      // Missing or replaced by a newer host: leave it alone.
    }
  }

  const publishBridge = async (port) => {
    await mkdir(dirname(bridgePath), { recursive: true, mode: 0o700 })
    const temporary = `${bridgePath}.${process.pid}.tmp`
    const value = JSON.stringify({
      version: BRIDGE_VERSION,
      port,
      token,
      pid: process.pid,
    })
    await writeFile(temporary, value, { mode: 0o600 })
    await rename(temporary, bridgePath)
    await chmod(bridgePath, 0o600).catch(() => {})
  }

  const openWorkspace = async (rawBody) => {
    const request = JSON.parse(rawBody)
    if (typeof request !== 'object' || request === null || typeof request.path !== 'string') {
      throw new Error('path is required')
    }
    const target = resolve(request.path)
    if (!isAbsolute(target)) throw new Error('path must be absolute')
    const info = await stat(target)
    if (!info.isDirectory()) throw new Error('path is not a directory')

    const existing = await ctx.workspaceRegistry.resolveByPath(target)
    const workspace = existing ?? await ctx.workspaceRegistry.create(target)
    const pendingEntry = pending.get(String(workspace.id))
    if (pendingEntry !== undefined && workspace.title.startsWith(MARKER_PREFIX)) {
      return {
        ok: true,
        workspaceId: String(workspace.id),
        created: false,
        requestId: pendingEntry.requestId,
      }
    }

    const previousTitle = cleanTitle(workspace.title) || basename(target)
    const supplied = typeof request.requestId === 'string' ? request.requestId : ''
    const requestId = /^[A-Za-z0-9_-]{1,64}$/u.test(supplied) ? supplied : randomUUID()
    await workspace.setTitle(markerTitle(requestId, previousTitle))
    pending.set(String(workspace.id), { requestId, previousTitle })

    return {
      ok: true,
      workspaceId: String(workspace.id),
      created: existing === undefined,
      requestId,
    }
  }

  const removeWorkspace = async (rawBody) => {
    const request = JSON.parse(rawBody)
    if (typeof request !== 'object' || request === null || typeof request.path !== 'string') {
      throw new Error('path is required')
    }
    const target = resolve(request.path)
    const candidates = new Set([target])
    try {
      candidates.add(await realpath(target))
    } catch {
      // The worktree may already be deleted; canonicalize its parent instead.
      try {
        candidates.add(join(await realpath(dirname(target)), basename(target)))
      } catch {
        // Keep the resolved spelling only.
      }
    }

    for (const workspace of ctx.workspaceRegistry.list()) {
      if (!candidates.has(workspace.path)) continue
      await ctx.workspaceRegistry.delete(workspace.id)
      pending.delete(String(workspace.id))
      return { ok: true, removed: true, workspaceId: String(workspace.id) }
    }
    return { ok: true, removed: false }
  }

  const handle = async (req, res) => {
    try {
      const pathname = (req.url ?? '/').split('?')[0]
      if (req.method === 'GET' && pathname === '/health') {
        send(res, 200, { ok: true, version: BRIDGE_VERSION, pid: process.pid })
        return
      }
      if (req.method !== 'POST' || (pathname !== '/open' && pathname !== '/remove')) {
        send(res, 404, { ok: false, error: 'not found' })
        return
      }
      if (req.headers.authorization !== `Bearer ${token}`) {
        send(res, 401, { ok: false, error: 'unauthorized' })
        return
      }
      const body = await readBody(req)
      send(res, 200, pathname === '/open' ? await openWorkspace(body) : await removeWorkspace(body))
    } catch (error) {
      send(res, 400, {
        ok: false,
        error: error instanceof Error ? error.message : String(error),
      })
    }
  }

  const start = async () => {
    server = createServer((req, res) => {
      handle(req, res).catch((error) => {
        console.error('[qg-dsh-open] request failed:', error)
        if (!res.headersSent) send(res, 500, { ok: false, error: 'internal error' })
      })
    })
    await new Promise((resolveListen, rejectListen) => {
      server.once('error', rejectListen)
      server.listen(0, '127.0.0.1', resolveListen)
    })
    const address = server.address()
    if (typeof address !== 'object' || address === null) throw new Error('bridge has no address')
    await publishBridge(address.port)
    console.info(`[qg-dsh-open] bridge listening on 127.0.0.1:${address.port}`)
  }

  const stop = async () => {
    if (stopped) return
    stopped = true
    pending.clear()
    if (server !== undefined) {
      const closing = server
      server = undefined
      await new Promise((resolveClose) => {
        closing.close(() => resolveClose())
        closing.closeAllConnections?.()
      })
    }
    await removeBridgeFile()
  }

  ctx.effect(() => {
    const ready = start().catch((error) => {
      console.error('[qg-dsh-open] bridge failed to start:', error)
    })
    return async () => {
      await ready
      await stop()
    }
  })
}
