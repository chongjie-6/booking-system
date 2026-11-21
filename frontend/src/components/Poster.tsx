import type { CSSProperties } from "react";
import { hue, posterURL, type Movie } from "@/lib/data";

export function Poster({ movie }: { movie: Movie }) {
  return (
    <div
      className="poster"
      style={{ "--hue": hue(movie.movieID) } as CSSProperties}
      aria-hidden="true"
    >
      {movie.posterPath ? (
        <img src={posterURL(movie.posterPath)} alt="" loading="lazy" />
      ) : (
        <span>{movie.title}</span>
      )}
    </div>
  );
}

export function TmdbCredit() {
  return (
    <footer className="credits">
      Film details and posters from{" "}
      <a className="link" href="https://www.themoviedb.org">
        TMDB
      </a>
      . This product uses the TMDB API but is not endorsed or certified by TMDB.
    </footer>
  );
}
