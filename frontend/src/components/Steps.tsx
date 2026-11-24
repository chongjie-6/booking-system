const STEPS = ["Seats", "Payment", "Confirmed"];

// Where the user is in booking, so they know what comes next.
export function Steps({ at }: { at: number }) {
  return (
    <ol className="steps" aria-label="Booking progress">
      {STEPS.map((s, i) => (
        <li key={s} aria-current={i === at ? "step" : undefined}>
          {s}
        </li>
      ))}
    </ol>
  );
}
