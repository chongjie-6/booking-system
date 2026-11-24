import { describe, expect, it } from "vitest";
import {
  canCancel,
  distanceKm,
  formatKm,
  findSessions,
  findShowtime,
  fromParams,
  seatLabel,
  showtimesFor,
  slug,
  splitBookings,
  toLocalInput,
  toParams,
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
  totalCents: 1500,
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

  it("finds sessions by cinema, film, day and time of day", () => {
    const at = (eventID: string, startsAt: string, cinemaID: string) =>
      ({ eventID, startsAt, format: "Standard", room: "Screen 1", cinemaID, cinema: cinemaID });
    const films = [
      movie([at("past", "2026-06-01T08:00:00", "c1"), at("am", "2026-06-01T10:00:00", "c1"), at("pm", "2026-06-01T19:00:00", "c2")]),
      { ...movie([at("next", "2026-06-02T13:00:00", "c1")]), movieID: "m2" },
    ];
    const now = new Date("2026-06-01T09:00:00");
    const ids = (q: Partial<Parameters<typeof findSessions>[1]>) =>
      findSessions(films, { cinemas: [], movies: [], days: [], times: [], ...q }, now).map((s) => s.id);
    expect(ids({})).toEqual(["am", "pm", "next"]);
    expect(ids({ cinemas: ["c1"] })).toEqual(["am", "next"]);
    expect(ids({ movies: ["m2"] })).toEqual(["next"]);
    expect(ids({ days: ["2026-06-01"], times: ["evening"] })).toEqual(["pm"]);
    expect(ids({ cinemas: ["c2"], times: ["morning"] })).toEqual([]);
  });

  it("measures distance between places", () => {
    const melbourne = { lat: -37.8102, lng: 144.9628 };
    const mildura = { lat: -34.1855, lng: 142.161 };
    expect(distanceKm(melbourne, melbourne)).toBe(0);
    expect(distanceKm(melbourne, mildura)).toBeCloseTo(475.3, 0); // as the crow flies; about 540 km by road
    expect(distanceKm({ lat: 0, lng: 179.5 }, { lat: 0, lng: -179.5 })).toBeCloseTo(111.2, 0); // across the date line
    expect(formatKm(0.4)).toBe("Under 1 km");
    expect(formatKm(1234.4)).toBe("1,234 km");
  });

  it("keeps a session search in the URL", () => {
    const q = { cinemas: ["a", "b"], movies: [], days: ["2026-06-01"], times: ["evening" as const] };
    expect(toParams(q).toString()).toBe("cinemas=a%2Cb&days=2026-06-01&times=evening");
    expect(fromParams(toParams(q))).toEqual(q);
    expect(fromParams(new URLSearchParams("times=evening,toString,lunch")).times).toEqual(["evening"]);
  });

  it("finds showtimes by event and by day", () => {
    const m = movie([
      { eventID: "a", startsAt: "2026-06-01T09:00:00", format: "IMAX", room: "Screen 1", cinemaID: "c1", cinema: "Mildura" },
      { eventID: "b", startsAt: "2026-06-02T09:00:00", format: "IMAX", room: "Screen 2", cinemaID: "c1", cinema: "Mildura" },
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

  it("round-trips a time through a datetime-local value", () => {
    const iso = new Date(2026, 10, 24, 19, 30).toISOString();
    expect(toLocalInput(iso)).toBe("2026-11-24T19:30");
    expect(new Date(toLocalInput(iso)).toISOString()).toBe(iso);
  });

  it("formats seats and totals", () => {
    expect(seatLabel({ row: "C", number: 4 })).toBe("C4");
    expect(total([{ priceCents: 1550 }, { priceCents: 1550 }, { priceCents: 1850 }])).toBe("$49.50");
  });
});
