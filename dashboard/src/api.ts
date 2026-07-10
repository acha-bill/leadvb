async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    credentials: 'include',
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new ApiError(res.status, (data as { error?: string }).error || res.statusText)
  }
  return data as T
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
}

export function connectDashboardWS(onEvent: (ev: { type: string; [k: string]: unknown }) => void): () => void {
  let ws: WebSocket | null = null
  let closed = false
  let retry = 0

  function connect() {
    if (closed) return
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    ws = new WebSocket(`${proto}://${location.host}/ws/dashboard`)
    ws.onopen = () => {
      retry = 0
    }
    ws.onmessage = (e) => {
      try {
        onEvent(JSON.parse(e.data))
      } catch {
        /* ignore */
      }
    }
    ws.onclose = () => {
      ws = null
      if (!closed) {
        retry++
        setTimeout(connect, Math.min(15000, 1000 * retry))
      }
    }
  }
  connect()
  return () => {
    closed = true
    ws?.close()
  }
}
