import { useEffect, useState } from "react";
import { Link } from "react-router";
import { api } from "@/lib/api";
import { formatDay, formatRuntime, type Movie } from "@/lib/data";
import { Poster } from "@/components/Poster";
import { Carousel } from "@/components/Carousel";
import { Hero } from "@/components/Hero";
import { MoviesStatus } from "@/components/MoviesStatus";
import { TmdbCredit } from "@/components/Poster";
import { useTitle } from "@/lib/routes";

export default function Home({
  movies,
  onRetry,
}: {
  movies: Movie[] | null | undefined;
  onRetry: () => void;
}) {
  useTitle("Now showing");
  // Nice to have, so it stays hidden while loading or if it fails.
  const [soon, setSoon] = useState<Movie[]>([]);
  useEffect(() => {
    api.get<Movie[]>("/movies/coming-soon").then(setSoon, () => {});
  }, []);

  return (
    <>
      <Hero movies={movies?.filter((m) => m.featured) ?? []} />
      <main>
        <div className="section-head">
          <h1 tabIndex={-1}>Now showing</h1>
          <Link className="link" to="/movies">
            See all ›
          </Link>
        </div>
        <MoviesStatus movies={movies} onRetry={onRetry} />
        {movies?.length === 0 && (
          <p className="none">Nothing is showing in the next few days.</p>
        )}
        {!!movies?.length && <Carousel movies={movies} />}
        {!!soon.length && (
          <section className="soon">
            <div className="section-head">
              <h2>Coming soon</h2>
            </div>
            <ul className="carousel">
              {soon.map((movie) => (
                <li key={movie.movieID}>
                  <Poster movie={movie} />
                  <h3>{movie.title}</h3>
                  <p className="meta">
                    <span className="rating">{movie.rating}</span>
                    <span>{formatRuntime(movie.runtimeMinutes)}</span>
                    <span>From {formatDay(new Date(movie.showings[0].startsAt))}</span>
                  </p>
                </li>
              ))}
            </ul>
          </section>
        )}
      </main>
      {[...(movies ?? []), ...soon].some((m) => m.posterPath) && <TmdbCredit />}
    </>
  );
}
