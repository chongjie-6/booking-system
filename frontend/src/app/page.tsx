import { Link } from "react-router";
import { savedCinemas, type Movie } from "@/lib/data";
import { Carousel } from "@/components/Carousel";
import { Hero } from "@/components/Hero";
import { MoviesStatus } from "@/components/MoviesStatus";
import { QuickBook } from "@/components/QuickBook";
import { TmdbCredit } from "@/components/Poster";
import { useTitle } from "@/lib/routes";

export default function Home({
  movies,
  soon,
  onRetry,
}: {
  movies: Movie[] | null | undefined;
  soon: Movie[];
  onRetry: () => void;
}) {
  useTitle("Now showing");

  return (
    <>
      <Hero movies={movies?.filter((m) => m.featured) ?? []} />
      <QuickBook movies={movies} initial={{ cinemas: savedCinemas(), movies: [], days: [], times: [] }} />
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
            <Carousel movies={soon} soon />
          </section>
        )}
      </main>
      {[...(movies ?? []), ...soon].some((m) => m.posterPath) && <TmdbCredit />}
    </>
  );
}
