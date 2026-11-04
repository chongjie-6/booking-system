import { useState, type FormEvent } from 'react'
import { api, ApiError, type User } from './api'

export type AuthMode = 'login' | 'register' | 'forgot' | 'reset'

const copy: Record<AuthMode, { title: string; submit: string }> = {
  login: { title: 'Sign in', submit: 'Sign in' },
  register: { title: 'Create an account', submit: 'Create account' },
  forgot: { title: 'Reset your password', submit: 'Email me a link' },
  reset: { title: 'Choose a new password', submit: 'Save password' },
}

// Sign in, register, and both halves of a password reset. resetToken comes from the emailed link.
export function Auth({
  mode: startMode = 'login',
  resetToken,
  onSignedIn,
}: {
  mode?: AuthMode
  resetToken?: string
  onSignedIn: (user: User) => void
}) {
  const [mode, setMode] = useState(startMode)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [busy, setBusy] = useState(false)

  function switchTo(next: AuthMode) {
    setMode(next)
    setError('')
    setNotice('')
  }

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const f = Object.fromEntries(new FormData(e.currentTarget)) as Record<string, string>
    setBusy(true)
    setError('')
    setNotice('')
    try {
      if (mode === 'login') {
        await api('/auth/login', { method: 'POST', body: { email: f.email, password: f.password } })
        onSignedIn(await api<User>('/user'))
      } else if (mode === 'register') {
        onSignedIn(await api<User>('/auth/register', { method: 'POST', body: f }))
      } else if (mode === 'forgot') {
        await api('/auth/password/forgot', { method: 'POST', body: { email: f.email } })
        setNotice("If that email has an account, we've sent it a link to reset the password.")
      } else {
        await api('/auth/password/reset', { method: 'POST', body: { token: resetToken, newPassword: f.newPassword } })
        switchTo('login')
        setNotice('Password changed. Sign in with your new password.')
      }
    } catch (err) {
      setError(err instanceof ApiError && err.status === 401 ? 'Wrong email or password.' : (err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const newPassword = { type: 'password', required: true, minLength: 8, maxLength: 72, autoComplete: 'new-password' }

  return (
    <form className="fields" onSubmit={submit}>
      <h2>{copy[mode].title}</h2>
      {mode === 'register' && (
        <>
          <label>
            First name
            <input name="firstName" required autoComplete="given-name" />
          </label>
          <label>
            Last name
            <input name="lastName" required autoComplete="family-name" />
          </label>
        </>
      )}
      {mode !== 'reset' && (
        <label>
          Email
          <input name="email" type="email" required autoComplete="email" />
        </label>
      )}
      {mode === 'login' && (
        <label>
          Password
          <input name="password" type="password" required autoComplete="current-password" />
        </label>
      )}
      {mode === 'register' && (
        <label>
          Password
          <input name="password" {...newPassword} />
        </label>
      )}
      {mode === 'reset' && (
        <label>
          New password
          <input name="newPassword" {...newPassword} />
        </label>
      )}
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {notice && (
        <p className="hint" role="status">
          {notice}
        </p>
      )}
      <button className="primary" disabled={busy}>
        {copy[mode].submit}
      </button>
      <p className="switch">
        {mode === 'login' ? (
          <>
            <button type="button" className="link" onClick={() => switchTo('register')}>
              Create an account
            </button>
            <button type="button" className="link" onClick={() => switchTo('forgot')}>
              Forgot your password?
            </button>
          </>
        ) : (
          <button type="button" className="link" onClick={() => switchTo('login')}>
            Back to sign in
          </button>
        )}
      </p>
    </form>
  )
}
