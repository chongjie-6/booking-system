import { describe, expect, it } from "vitest";
import {
  canCancel,
  findShowtime,
  seatLabel,
  showtimesFor,
  slug,
  splitBookings,
  total,
  type Movie,
  type UserBooking,
} from "./data";

const movie = (showings: Movie["showings"]): Movie => ({
  movieID: "m1",
  title: "Film",
  genre: "Drama",
  runtimeMinutes: 90,
  rating: "M",
  synopsis: "",
  featured: false,
  posterPath: "",
  backdropPath: "",
  trailerKey: "",
  showings,
});

const booking = (ref: string, startsAt: string): UserBooking => ({
  bookingID: ref,
  ref,
  eventID: "e",
  title: "Film",
  startsAt,
  room: "Screen 1",
  priceCents: 1500,
  seats: [{ row: "C", number: 4 }],
});

describe("data", () => {
  it("makes URL slugs from titles", () => {
    expect(slug("Spider-Man: Into the Spider-Verse")).toBe("spider-man-into-the-spider-verse");
    expect(slug("  WALL·E!  ")).toBe("wall-e");
  });

  it("splits bookings into upcoming (soonest first) and past (latest first)", () => {
    const now = Date.parse("2026-06-01T12:00:00Z");
    const fromApi = [
      booking("LATER", "2026-06-03T10:00:00Z"),
      booking("SOON", "2026-06-02T10:00:00Z"),
      booking("RECENT", "2026-05-31T10:00:00Z"),
      booking("OLD", "2026-05-01T10:00:00Z"),
    ];
    const { upcoming, past } = splitBookings(fromApi, now);
    expect(upcoming.map((b) => b.ref)).toEqual(["SOON", "LATER"]);
    expect(past.map((b) => b.ref)).toEqual(["RECENT", "OLD"]);
    expect(fromApi[0].ref).toBe("LATER"); // the API's list isn't reordered in place
  });

  it("finds showtimes by event and by day", () => {
    const m = movie([
      { eventID: "a", startsAt: "2026-06-01T09:00:00", priceCents: 1500, room: "Screen 1" },
      { eventID: "b", startsAt: "2026-06-02T09:00:00", priceCents: 1500, room: "Screen 2" },
    ]);
    expect(findShowtime([m], "b")?.room).toBe("Screen 2");
    expect(findShowtime([m], "missing")).toBeUndefined();
    expect(showtimesFor(m, new Date(2026, 5, 1)).map((s) => s.id)).toEqual(["a"]);
  });

  it("allows cancelling until an hour before the showing", () => {
    const now = Date.parse("2026-06-01T12:00:00Z");
    expect(canCancel({ startsAt: "2026-06-01T13:00:01Z" }, now)).toBe(true);
    expect(canCancel({ startsAt: "2026-06-01T13:00:00Z" }, now)).toBe(false);
    expect(canCancel({ startsAt: "2026-05-31T13:00:00Z" }, now)).toBe(false);
  });

  it("formats seats and totals", () => {
    expect(seatLabel({ row: "C", number: 4 })).toBe("C4");
    expect(total({ priceCents: 1550 }, 3)).toBe("$46.50");
  });
});
