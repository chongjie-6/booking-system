import { useRef } from "react";
import { Link } from "react-router";
import { formatRuntime, slug, type Movie } from "@/lib/data";
import { Poster } from "@/components/Poster";

export function Carousel({ movies }: { movies: Movie[] }) {
  const track = useRef<HTMLUListElement>(null);
  const page = (dir: number) =>
    track.current?.scrollBy({ left: dir * track.current.clientWidth });

  return (
    <>
      <ul ref={track} className="carousel">
        {movies.map((movie) => (
          <li key={movie.movieID}>
            <Link
              to={`/movies/${slug(movie.title)}`}
              tabIndex={-1}
              aria-hidden="true"
            >
              <Poster movie={movie} />
            </Link>
            <h2>
              <Link to={`/movies/${slug(movie.title)}`}>{movie.title}</Link>
            </h2>
            <p className="meta">
              <span className="rating">{movie.rating}</span>
              <span>{formatRuntime(movie.runtimeMinutes)}</span>
            </p>
          </li>
        ))}
      </ul>
      <div className="carousel-nav">
        <button aria-label="Previous films" onClick={() => page(-1)}>
          ←
        </button>
        <button aria-label="Next films" onClick={() => page(1)}>
          →
        </button>
      </div>
    </>
  );
}
