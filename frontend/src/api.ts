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
export async function api<T = unknown>(path: string, { method = 'GET', body }: { method?: string; body?: unknown } = {}): Promise<T> {
  const send = () =>
    fetch(path, {
      method,
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
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
