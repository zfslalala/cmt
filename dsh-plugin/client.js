/**
 * Client half of the qg workspace bridge.
 *
 * The Host half creates the requested Workspace and temporarily prefixes its
 * title with a request marker. This module watches the Workspace list, opens
 * the marked Workspace, then restores the original title.
 */
window.__ModuleLoader__.load({
  id: 'qg-dsh-open',
  factory(require) {
    const MARKER_RE = /^qg-open:([^:]+):(.*)$/u

    function parseMarker(title) {
      const match = MARKER_RE.exec(title)
      if (match === null) return undefined
      try {
        return { requestId: match[1], title: decodeURIComponent(match[2]) }
      } catch {
        return undefined
      }
    }

    return {
      inject: ['workspaces', 'uiWorkspace'],
      apply(ctx) {
        const opened = new Set()
        let disposed = false

        const check = () => {
          if (disposed) return
          const snapshot = ctx.workspaces.list.getSnapshot()
          for (const workspace of snapshot.items) {
            const marker = parseMarker(workspace.title)
            if (marker === undefined || opened.has(marker.requestId)) continue
            opened.add(marker.requestId)
            Promise.resolve()
              .then(() => ctx.uiWorkspace.openWorkspace(workspace.workspaceId))
              .then(() => {
                if (disposed) return
                return ctx.workspaces.rename(workspace.workspaceId, marker.title)
              })
              .then(() => {
                console.info(`[qg-dsh-open] opened workspace ${workspace.path}`)
              })
              .catch((error) => {
                console.error('[qg-dsh-open] failed to open workspace:', error)
              })
          }
        }

        const unsubscribe = ctx.workspaces.list.subscribe(check)
        ctx.effect(() => () => {
          disposed = true
          unsubscribe()
        })
        check()
      },
    }
  },
})
