import type { Movie } from "@/lib/data";

export function MoviesStatus({
  movies,
  onRetry,
}: {
  movies: Movie[] | null | undefined;
  onRetry: () => void;
}) {
  if (movies === undefined)
    return (
      <p className="none" role="status">
        Loading films…
      </p>
    );
  if (movies === null)
    return (
      <p className="error" role="alert">
        Couldn't load the films.{" "}
        <button className="link" onClick={onRetry}>
          Try again
        </button>
      </p>
    );
  return null;
}
