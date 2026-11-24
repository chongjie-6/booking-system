import { useEffect, useRef, type ReactNode } from "react";

// A modal panel that slides in from the right, or with className="modal" rises in the middle. Closes on Esc, the close button,
// or a click on the backdrop.
export function Drawer({
  open,
  onClose,
  className = "",
  children,
}: {
  open: boolean;
  className?: string;
  onClose: () => void;
  children: ReactNode;
}) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    if (open) ref.current?.showModal();
    else ref.current?.close();
  }, [open]);

  return (
    <dialog
      ref={ref}
      className={`drawer ${className}`}
      onClose={onClose}
      onClick={(e) => e.target === e.currentTarget && ref.current?.close()}
    >
      <div className="drawer-body">
        <button
          className="close"
          aria-label="Close"
          onClick={() => ref.current?.close()}
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" aria-hidden="true">
            <path d="M6 6l12 12M18 6L6 18" />
          </svg>
        </button>
        {children}
      </div>
    </dialog>
  );
}
