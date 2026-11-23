import { afterEach, describe, expect, it, vi } from "vitest";
import { api, ApiError } from "./api";

const json = (status: number, body: unknown) => new Response(JSON.stringify(body), { status });

// Fakes the backend: /user needs a fresh session, which /auth/refresh hands out unless refreshOk is false.
function fakeServer({ refreshOk = true } = {}) {
  let fresh = false;
  const refreshes = vi.fn();
  vi.stubGlobal(
    "fetch",
    vi.fn(async (path: string) => {
      if (path === "/auth/refresh") {
        refreshes();
        fresh = refreshOk;
        return json(refreshOk ? 200 : 401, {});
      }
      if (path === "/auth/login") return json(401, { error: { message: "invalid credentials" } });
      if (path === "/busy") return json(429, {});
      return fresh ? json(200, { data: { firstName: "Ada" } }) : json(401, { error: { message: "signed out" } });
    }),
  );
  return refreshes;
}

afterEach(() => vi.unstubAllGlobals());

describe("api", () => {
  it("refreshes an expired session once and retries", async () => {
    const refreshes = fakeServer();
    await expect(api.get("/user")).resolves.toEqual({ firstName: "Ada" });
    expect(refreshes).toHaveBeenCalledTimes(1);
  });

  it("shares one refresh between calls that expire together, since refresh tokens are single-use", async () => {
    const refreshes = fakeServer();
    await Promise.all([api.get("/user"), api.get("/user"), api.get("/user")]);
    expect(refreshes).toHaveBeenCalledTimes(1);
  });

  it("treats a 401 from login as a wrong password, not an expired session", async () => {
    const refreshes = fakeServer();
    await expect(api.post("/auth/login", {})).rejects.toMatchObject({ status: 401 });
    expect(refreshes).not.toHaveBeenCalled();
  });

  it("throws the server's message when the session is over", async () => {
    fakeServer({ refreshOk: false });
    const err = await api.get("/user").catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 401, message: "signed out" });
  });

  it("explains rate limits in words", async () => {
    fakeServer();
    await expect(api.get("/busy")).rejects.toThrow("Too many attempts");
  });
});
