import { useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import { backdropURL, slug, type Movie } from "@/lib/data";

const reducedMotion = matchMedia("(prefers-reduced-motion: reduce)").matches;
// How far a swipe has to travel before it turns the slide, rather than springing back.
const SWIPE_PX = 50;

// Full-width banner for the featured films: a muted trailer playing behind the text, else the backdrop.
// With several films it swipes like the carousel, looping past either end.
export function Hero({ movies }: { movies: Movie[] }) {
  const [i, setI] = useState(0);
  const ref = useRef<HTMLElement>(null);
  // True once the banner has scrolled up behind the masthead, which then turns solid.
  const [past, setPast] = useState(false);
  // How far the slides are being dragged, and the slide (-1, 0 or 1) they ease to once let go.
  const [dx, setDx] = useState(0);
  const [settle, setSettle] = useState<number | null>(null);
  const drag = useRef({ active: false, moved: false, x: 0 });
  const n = movies.length;
  const loop = n > 1;
  const at = (k: number) => movies[(((i + k) % n) + n) % n];
  const shown = n > 0;

  useEffect(() => {
    if (!ref.current) return;
    const masthead = document.querySelector(".masthead")?.clientHeight ?? 0;
    const io = new IntersectionObserver(([e]) => setPast(!e.isIntersecting), {
      rootMargin: `-${masthead}px 0px 0px`,
    });
    io.observe(ref.current);
    return () => io.disconnect();
  }, [shown]);

  if (!shown) return null;

  const finish = (dir: number) => {
    setI((j) => (((j + dir) % n) + n) % n);
    setDx(0);
    setSettle(null);
  };

  const release = () => {
    const d = drag.current;
    if (!d.active) return;
    d.active = false;
    if (!d.moved) return;
    const dir = dx < -SWIPE_PX ? 1 : dx > SWIPE_PX ? -1 : 0;
    // No transition runs (so no transitionend) when nothing would move.
    if (reducedMotion || (!dir && !dx)) finish(dir);
    else setSettle(dir);
  };

  return (
    <section
      ref={ref}
      className="hero"
      aria-label="Featured"
      aria-roledescription="carousel"
      data-past={past || undefined}
    >
      <div
        className={settle === null ? "hero-track" : "hero-track settling"}
        style={
          loop
            ? {
                translate:
                  settle === null
                    ? `calc(-100% + ${dx}px)`
                    : `${-100 - 100 * settle}%`,
              }
            : undefined
        }
        onTransitionEnd={(e) => {
          if (e.target === e.currentTarget && settle !== null) finish(settle);
        }}
        onDragStart={(e) => e.preventDefault()}
        onPointerDown={(e) => {
          if (!loop || settle !== null || e.button !== 0) return;
          drag.current = { active: true, moved: false, x: e.clientX };
        }}
        onPointerMove={(e) => {
          const d = drag.current;
          const move = e.clientX - d.x;
          if (!d.active || (!d.moved && Math.abs(move) < 5)) return;
          if (!d.moved) {
            d.moved = true;
            e.currentTarget.setPointerCapture(e.pointerId);
          }
          setDx(move);
        }}
        onPointerUp={release}
        onPointerCancel={release}
        onClickCapture={(e) => {
          if (drag.current.moved) e.preventDefault();
        }}
      >
        {(loop ? [-1, 0, 1] : [0]).map((k) => {
          const movie = at(k);
          const trailer = k === 0 && movie.trailerKey && !reducedMotion;
          return (
            <div key={k} className="hero-slide" inert={k !== 0}>
              {movie.backdropPath && (
                <img
                  className="hero-media"
                  src={backdropURL(movie.backdropPath)}
                  alt=""
                />
              )}
              {trailer && (
                <iframe
                  key={movie.trailerKey}
                  className="hero-media hero-video"
                  src={`https://www.youtube-nocookie.com/embed/${movie.trailerKey}?autoplay=1&mute=1&loop=1&playlist=${movie.trailerKey}&controls=0&disablekb=1&playsinline=1&rel=0`}
                  title={`${movie.title} trailer`}
                  allow="autoplay; encrypted-media"
                  tabIndex={-1}
                  aria-hidden="true"
                />
              )}
              <div className="hero-body">
                <p className="hero-tag">Featured</p>
                <h2>{movie.title}</h2>
                <p className="hero-synopsis">{movie.synopsis}</p>
                <Link className="primary" to={`/movies/${slug(movie.title)}`}>
                  Book now
                </Link>
              </div>
            </div>
          );
        })}
      </div>
      {loop && (
        <div className="hero-body hero-nav">
          <div className="hero-dots">
            {movies.map((m, j) => (
              <button
                key={m.movieID}
                aria-label={`Show ${m.title}`}
                aria-pressed={j === i}
                onClick={() => settle === null && setI(j)}
              />
            ))}
          </div>
        </div>
      )}
    </section>
  );
}
