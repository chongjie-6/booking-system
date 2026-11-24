// @vitest-environment jsdom
import { afterEach, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import Done from "./page";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

const open = () =>
  render(
    <MemoryRouter initialEntries={["/bookings/AB12CD34"]}>
      <Routes>
        <Route path="/bookings/:ref" element={<Done />} />
      </Routes>
    </MemoryRouter>,
  );

it("shows the ticket for a booking opened from a link", async () => {
  const fetch = vi.fn(async () =>
    Response.json({
      data: { bookingID: "b", ref: "AB12CD34", eventID: "e", title: "Toy Story", startsAt: "2026-06-01T09:00:00Z", room: "Screen 2", totalCents: 1500, seats: [{ row: "C", number: 4 }] },
    }),
  );
  vi.stubGlobal("fetch", fetch);
  open();
  expect(await screen.findByRole("heading", { name: "Toy Story" })).toBeTruthy();
  expect(screen.getByText("C4")).toBeTruthy();
  expect(fetch).toHaveBeenCalledWith("/bookings/AB12CD34", expect.anything());
});

it("falls back to the reference when the booking can't be fetched", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => Response.json({}, { status: 401 })));
  open();
  expect(await screen.findByText(/Show reference AB12CD34 at the door/)).toBeTruthy();
});
