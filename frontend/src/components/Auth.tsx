import { useId, useState, type FormEvent, type InputHTMLAttributes } from "react";
import { api, ApiError, type User } from "@/lib/api";

export type AuthMode = "login" | "register" | "forgot" | "reset";

const copy: Record<AuthMode, { title: string; submit: string }> = {
  login: { title: "Sign in", submit: "Sign in" },
  register: { title: "Create an account", submit: "Create account" },
  forgot: { title: "Reset your password", submit: "Email me a link" },
  reset: { title: "Choose a new password", submit: "Save password" },
};

// Sign in, register, and both halves of a password reset. resetToken comes from the emailed link.
export function Auth({
  mode: startMode = "login",
  resetToken,
  onSignedIn,
}: {
  mode?: AuthMode;
  resetToken?: string;
  onSignedIn: (user: User) => void;
}) {
  const [mode, setMode] = useState(startMode);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const ruleId = useId();

  function switchTo(next: AuthMode) {
    setMode(next);
    setError("");
    setNotice("");
  }

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = Object.fromEntries(new FormData(e.currentTarget)) as Record<
      string,
      string
    >;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      if (mode === "login") {
        await api.post("/auth/login", { email: f.email, password: f.password });
        onSignedIn(await api.get<User>("/user"));
      } else if (mode === "register") {
        onSignedIn(
          await api.post<User>("/auth/register", f),
        );
      } else if (mode === "forgot") {
        await api.post("/auth/password/forgot", { email: f.email });
        setNotice(
          "If that email has an account, we've sent it a link to reset the password.",
        );
      } else {
        await api.post("/auth/password/reset", { token: resetToken, newPassword: f.newPassword });
        switchTo("login");
        setNotice("Password changed. Sign in with your new password.");
      }
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 401
          ? "Wrong email or password."
          : (err as Error).message,
      );
    } finally {
      setBusy(false);
    }
  }

  const newPassword = {
    required: true,
    minLength: 8,
    maxLength: 72,
    autoComplete: "new-password",
    "aria-describedby": ruleId,
  };

  return (
    <form className="fields" onSubmit={submit}>
      <h2 tabIndex={-1}>{copy[mode].title}</h2>
      {mode === "register" && (
        <div className="pair">
          <Field
            label="First name"
            name="firstName"
            required
            autoComplete="given-name"
          />
          <Field
            label="Last name"
            name="lastName"
            required
            autoComplete="family-name"
          />
        </div>
      )}
      <div className="group">
        {mode !== "reset" && (
          <Field
            label="Email"
            name="email"
            type="email"
            required
            autoComplete="email"
          />
        )}
        {mode === "login" && (
          <Field
            label="Password"
            name="password"
            type="password"
            required
            autoComplete="current-password"
          />
        )}
        {(mode === "register" || mode === "reset") && (
          <div>
            <Field
              label={mode === "reset" ? "New password" : "Password"}
              name={mode === "reset" ? "newPassword" : "password"}
              type="password"
              {...newPassword}
            />
            <p className="rule" id={ruleId}>
              At least 8 characters.
            </p>
          </div>
        )}
      </div>
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
      <div className="actions">
        <button className="primary" disabled={busy}>
          {busy ? "Please wait…" : copy[mode].submit}
        </button>
        {mode === "login" && (
          <button
            type="button"
            className="secondary"
            onClick={() => switchTo("forgot")}
          >
            Forgot your password?
          </button>
        )}
      </div>
      {mode === "login" ? (
        <div className="switch">
          <button
            type="button"
            className="secondary"
            onClick={() => switchTo("register")}
          >
            Create an account
          </button>
        </div>
      ) : (
        <p>
          <button
            type="button"
            className="link"
            onClick={() => switchTo("login")}
          >
            Back to sign in
          </button>
        </p>
      )}
    </form>
  );
}

// The label sits inside the input like a placeholder, then shrinks above the text once the input has focus or a value.
export function Field({
  label,
  type,
  ...props
}: { label: string } & InputHTMLAttributes<HTMLInputElement>) {
  const [shown, setShown] = useState(false);
  const password = type === "password";
  return (
    <label className="float">
      <span>{label}</span>
      <input {...props} type={password && shown ? "text" : type} placeholder=" " />
      {password && (
        <button
          type="button"
          className="link"
          aria-label={shown ? "Hide password" : "Show password"}
          onClick={() => setShown(!shown)}
        >
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            aria-hidden="true"
          >
            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
            <circle cx="12" cy="12" r="3" />
            {shown && <path d="M3 3l18 18" />}
          </svg>
        </button>
      )}
    </label>
  );
}
