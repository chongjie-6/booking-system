import { useState, type FormEvent } from "react";
import { ApiError } from "@/lib/api";

// Submits a form's fields, tracking busy, error and success states. messages maps a status to friendlier text.
export function useForm(messages: Record<number, string>) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  async function run(e: FormEvent<HTMLFormElement>, send: (f: Record<string, string>) => Promise<string>) {
    e.preventDefault();
    const form = e.currentTarget;
    const f = Object.fromEntries(new FormData(form)) as Record<string, string>;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      setNotice(await send(f));
    } catch (err) {
      setError((err instanceof ApiError && messages[err.status]) || (err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return { busy, error, notice, run };
}
