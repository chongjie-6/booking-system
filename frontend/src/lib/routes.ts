import { useEffect } from "react";
import type { Booking, Seat } from "./data";

// What each history entry keeps, so Back, Forward and refresh return with the same seats.
// pushed marks entries the app added, which a back button can safely return from.
export type Entry = { seats?: Seat[]; booking?: Booking; pushed?: boolean };

export function useTitle(title: string) {
  useEffect(() => {
    document.title = `${title} · Lantern Picture House`;
  }, [title]);
}
