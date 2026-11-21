import { Link } from "react-router";
import type { Movie } from "@/lib/data";
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
      </main>
      {movies?.some((m) => m.posterPath) && <TmdbCredit />}
    </>
  );
}
