// @vitest-environment jsdom
import { afterEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import Account from "./page";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

const booking = (ref: string, inMinutes: number) => ({
  bookingID: `id-${ref}`,
  ref,
  eventID: "e",
  title: `Film ${ref}`,
  startsAt: new Date(Date.now() + inMinutes * 60_000).toISOString(),
  room: "Screen 1",
  priceCents: 1500,
  seats: [{ row: "A", number: 1 }],
});

it("offers cancelling only up to an hour before, and cancels after confirming", async () => {
  let bookings = [booking("SOON", 30), booking("LATER", 180)];
  const fetch = vi.fn(async (path: string, init?: RequestInit) => {
    if (init?.method === "DELETE") bookings = bookings.filter((b) => path !== `/bookings/${b.bookingID}`);
    return Response.json({ data: init?.method === "DELETE" ? null : bookings });
  });
  vi.stubGlobal("fetch", fetch);
  vi.stubGlobal("confirm", () => true);
  const user = { userID: "u", userRole: "user", firstName: "Ada", lastName: "L", email: "ada@example.com" };
  render(
    <MemoryRouter>
      <Account user={user} onSignedIn={() => {}} onUpdated={() => {}} />
    </MemoryRouter>,
  );

  await screen.findByRole("heading", { name: "Film LATER" });
  expect(screen.queryByRole("button", { name: "Cancel booking SOON" })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Cancel booking LATER" }));

  expect(await screen.findByText(/Your booking for Film LATER is cancelled/)).toBeTruthy();
  expect(fetch).toHaveBeenCalledWith("/bookings/id-LATER", expect.objectContaining({ method: "DELETE" }));
  await vi.waitFor(() => expect(screen.queryByRole("heading", { name: "Film LATER" })).toBeNull());
});
