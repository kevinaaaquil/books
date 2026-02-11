"use client";

import { useState, useRef, useEffect } from "react";

type ProfileMenuProps = {
  email: string;
  onLogout: () => void;
};

export function ProfileMenu({ email, onLogout }: ProfileMenuProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    if (open) {
      document.addEventListener("mousedown", handleClickOutside);
      return () => document.removeEventListener("mousedown", handleClickOutside);
    }
  }, [open]);

  const initial = email ? email.charAt(0).toUpperCase() : "?";

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-accent text-stone-900 font-semibold text-sm ring-2 ring-stone-200 dark:ring-stone-600 hover:ring-accent focus:outline-none focus:ring-2 focus:ring-accent"
        aria-label="Profile menu"
      >
        {initial}
      </button>
      {open && (
        <div className="absolute right-0 top-full z-20 mt-2 min-w-[200px] rounded-lg border border-stone-200 dark:border-stone-600 bg-white dark:bg-stone-800 shadow-lg py-2">
          <p className="px-3 py-2 text-sm text-stone-600 dark:text-stone-400 truncate border-b border-stone-100 dark:border-stone-700" title={email}>
            {email || "—"}
          </p>
          <button
            type="button"
            onClick={() => {
              setOpen(false);
              onLogout();
            }}
            className="w-full px-3 py-2 text-left text-sm font-medium text-stone-700 dark:text-stone-300 hover:bg-stone-100 dark:hover:bg-stone-700"
          >
            Log out
          </button>
        </div>
      )}
    </div>
  );
}
