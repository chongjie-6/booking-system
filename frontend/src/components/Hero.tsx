import { useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import { backdropURL, slug, type Movie } from "@/lib/data";

const reducedMotion = matchMedia("(prefers-reduced-motion: reduce)").matches;

// Full-width banner for the featured films: a muted trailer playing behind the text, else the backdrop.
export function Hero({ movies }: { movies: Movie[] }) {
  const [i, setI] = useState(0);
  const ref = useRef<HTMLElement>(null);
  // True once the banner has scrolled up behind the masthead, which then turns solid.
  const [past, setPast] = useState(false);
  const movie = movies[i] ?? movies[0];
  const shown = !!movie;

  useEffect(() => {
    if (!ref.current) return;
    const masthead = document.querySelector(".masthead")?.clientHeight ?? 0;
    const io = new IntersectionObserver(([e]) => setPast(!e.isIntersecting), {
      rootMargin: `-${masthead}px 0px 0px`,
    });
    io.observe(ref.current);
    return () => io.disconnect();
  }, [shown]);

  if (!movie) return null;
  const trailer = movie.trailerKey && !reducedMotion;

  return (
    <section
      ref={ref}
      className="hero"
      aria-label="Featured"
      data-past={past || undefined}
    >
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
        {movies.length > 1 && (
          <div className="hero-dots">
            {movies.map((m, j) => (
              <button
                key={m.movieID}
                aria-label={`Show ${m.title}`}
                aria-pressed={j === i}
                onClick={() => setI(j)}
              />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
