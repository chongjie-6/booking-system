import { useLayoutEffect, useRef } from "react";
import { Link } from "react-router";
import { formatDay, formatRuntime, slug, type Movie } from "@/lib/data";
import { Poster } from "@/components/Poster";

const SNAP_MS = 700;

// soon: the films are coming soon, so each shows its opening day and has an h3 to sit under the section's h2.
export function Carousel({ movies, soon = false }: { movies: Movie[]; soon?: boolean }) {
  const Heading = soon ? "h3" : "h2";
  const track = useRef<HTMLUListElement>(null);
  const drag = useRef({ active: false, moved: false, x: 0, left: 0 });
  const anim = useRef({ running: false, raf: 0, from: 0, to: 0 });
  const idle = useRef(0);
  const touching = useRef(false);
  // Enough copies that the viewport (at most ~5 cards wide) never runs past the last one.
  const copies = 2 + Math.ceil(5 / movies.length);

  const offset = (i: number) =>
    (track.current!.children[i] as HTMLElement).offsetLeft -
    (track.current!.children[0] as HTMLElement).offsetLeft;

  const stopSnap = () => {
    clearTimeout(idle.current);
    cancelAnimationFrame(anim.current.raf);
    anim.current.running = false;
  };

  // Eases to the nearest card over SNAP_MS, since CSS snapping has no duration control.
  const snap = () => {
    stopSnap();
    const el = track.current!;
    const a = anim.current;
    const step = offset(1);
    a.from = el.scrollLeft;
    a.to = Math.round(a.from / step) * step;
    a.running = true;
    const ms = matchMedia("(prefers-reduced-motion: reduce)").matches ? 0 : SNAP_MS;
    const start = performance.now();
    const frame = (now: number) => {
      const t = ms ? Math.min((now - start) / ms, 1) : 1;
      el.scrollLeft = a.from + (a.to - a.from) * (1 - (1 - t) ** 3);
      if (t < 1) a.raf = requestAnimationFrame(frame);
      else a.running = false;
    };
    a.raf = requestAnimationFrame(frame);
  };

  const snapWhenIdle = () => {
    clearTimeout(idle.current);
    idle.current = window.setTimeout(snap, 150);
  };

  useLayoutEffect(() => {
    track.current!.scrollTo({ left: offset(movies.length), behavior: "instant" });
    return stopSnap;
  }, [movies]);

  // Keeps the scroll position inside the second copy, jumping by one copy's width.
  const wrap = () => {
    const el = track.current!;
    const set = offset(movies.length);
    const shift =
      el.scrollLeft < set ? set : el.scrollLeft >= 2 * set ? -set : 0;
    if (!shift) return;
    el.scrollTo({ left: el.scrollLeft + shift, behavior: "instant" });
    drag.current.left += shift;
    anim.current.from += shift;
    anim.current.to += shift;
  };

  const endDrag = () => {
    const d = drag.current;
    if (!d.active) return;
    d.active = false;
    if (!d.moved) return;
    track.current!.classList.remove("dragging");
    snap();
  };

  return (
    <ul
      ref={track}
      className="carousel"
      onScroll={() => {
        wrap();
        if (!drag.current.active && !touching.current && !anim.current.running)
          snapWhenIdle();
      }}
      onWheel={stopSnap}
      onTouchStart={() => {
        touching.current = true;
        stopSnap();
      }}
      onTouchEnd={() => {
        touching.current = false;
        snapWhenIdle();
      }}
      onDragStart={(e) => e.preventDefault()}
      onPointerDown={(e) => {
        stopSnap();
        if (e.pointerType !== "mouse" || e.button !== 0) return;
        drag.current = {
          active: true,
          moved: false,
          x: e.clientX,
          left: track.current!.scrollLeft,
        };
      }}
      onPointerMove={(e) => {
        const d = drag.current;
        const el = track.current!;
        const dx = e.clientX - d.x;
        if (!d.active || (!d.moved && Math.abs(dx) < 5)) return;
        if (!d.moved) {
          d.moved = true;
          el.setPointerCapture(e.pointerId);
          el.classList.add("dragging");
        }
        el.scrollLeft = d.left - dx;
      }}
      onPointerUp={endDrag}
      onPointerCancel={endDrag}
      onClickCapture={(e) => {
        if (drag.current.moved) e.preventDefault();
      }}
    >
      {Array.from({ length: copies }, (_, copy) =>
        movies.map((movie) => (
          <li key={`${copy}-${movie.movieID}`} aria-hidden={copy !== 1}>
            <Link
              to={`/movies/${slug(movie.title)}`}
              tabIndex={-1}
              aria-hidden="true"
            >
              <Poster movie={movie} />
            </Link>
            <Heading>
              <Link
                to={`/movies/${slug(movie.title)}`}
                tabIndex={copy === 1 ? undefined : -1}
              >
                {movie.title}
              </Link>
            </Heading>
            <p className="meta">
              <span className="rating">{movie.rating}</span>
              <span>{formatRuntime(movie.runtimeMinutes)}</span>
              {soon && <span>From {formatDay(new Date(movie.showings[0].startsAt))}</span>}
            </p>
          </li>
        )),
      )}
    </ul>
  );
}
