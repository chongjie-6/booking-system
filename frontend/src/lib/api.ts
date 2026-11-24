export type User = { userID: string; userRole: string; firstName: string; lastName: string; email: string }

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

let refreshing: Promise<boolean> | null = null
let refreshes = 0 // successful refreshes so far

// Swaps the refresh cookie for new tokens. Refresh tokens are single-use, so calls that overlap share one request
// instead of each spending the same token. Resolves false when the session is over.
function refresh(): Promise<boolean> {
  refreshing ??= fetch('/auth/refresh', { method: 'POST' })
    .then((r) => (r.ok && refreshes++, r.ok), () => false)
    .finally(() => (refreshing = null))
  return refreshing
}

// Calls the API and returns the response's data. Access tokens last 15 minutes, so a 401 refreshes them and retries once;
// a 401 that survives that means the user is signed out. Throws ApiError with the server's message on failure.
async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const send = () =>
    fetch(path, {
      method,
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    }).catch(() => {
      // fetch only rejects when the request never got a response; its own message ("Failed to fetch") means nothing to users.
      throw new ApiError(0, "Couldn't reach the server. Check your connection and try again.")
    })

  const seen = refreshes
  let res = await send()
  // A 401 from login means a wrong password, not an expired token. If another call refreshed since this one was sent,
  // the new tokens are already in place, so just retry.
  if (res.status === 401 && path !== '/auth/login' && (refreshes !== seen || (await refresh()))) res = await send()

  const json = await res.json().catch(() => null)
  if (!res.ok) {
    if (res.status === 429) throw new ApiError(429, 'Too many attempts. Wait a minute and try again.')
    throw new ApiError(res.status, json?.error?.message ?? 'Something went wrong. Please try again.')
  }
  return json?.data as T
}

export const api = {
  get: <T = unknown>(path: string) => request<T>('GET', path),
  post: <T = unknown>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T = unknown>(path: string, body?: unknown) => request<T>('PUT', path, body),
  patch: <T = unknown>(path: string, body?: unknown) => request<T>('PATCH', path, body),
  del: <T = unknown>(path: string, body?: unknown) => request<T>('DELETE', path, body),
}

// Set when the user leaves checkout. The seat map waits for it, so the user's own seats don't come back as taken.
export let releasing: Promise<unknown> = Promise.resolve()

export function releaseHold(eventID: string, seatIDs: string[]) {
  releasing = api.del(`/events/${eventID}/hold`, { seatIDs }).catch(() => {})
}
