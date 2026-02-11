"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { login, loginAsGuest, setToken, setRole } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [guestLoading, setGuestLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const { token, role } = await login(email, password);
      setToken(token);
      if (role) setRole(role);
      router.push("/books");
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  async function handleViewAsGuest(e: React.MouseEvent) {
    e.preventDefault();
    setError("");
    setGuestLoading(true);
    try {
      const { token, role } = await loginAsGuest();
      setToken(token);
      if (role) setRole(role);
      router.push("/books");
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Guest access not available");
    } finally {
      setGuestLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-accent-soft dark:bg-accent-soft px-4">
      <div className="w-full max-w-sm rounded-xl border-2 border-accent/30 bg-white dark:bg-stone-800 shadow-lg shadow-accent/10 p-8">
        <h1 className="text-2xl font-semibold text-stone-900 dark:text-stone-100 mb-1">
          Sign in
        </h1>
        <p className="text-accent-muted text-sm mb-6">Welcome back to your library</p>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="email" className="block text-sm font-medium text-accent-muted mb-1">
              Email
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              className="w-full rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-3 py-2 text-stone-900 dark:text-stone-100 focus:outline-none focus:ring-2 focus:ring-accent"
              placeholder="you@example.com"
            />
          </div>
          <div>
            <label htmlFor="password" className="block text-sm font-medium text-accent-muted mb-1">
              Password
            </label>
            <div className="relative">
              <input
                id="password"
                type={showPassword ? "text" : "password"}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                className="w-full rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-3 py-2 pr-11 text-stone-900 dark:text-stone-100 focus:outline-none focus:ring-2 focus:ring-accent"
              />
              <button
                type="button"
                onClick={() => setShowPassword((s) => !s)}
                className="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 rounded text-white hover:bg-white/20 transition-colors"
                aria-label={showPassword ? "Hide password" : "Show password"}
              >
                {showPassword ? (
                  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                    <line x1="1" y1="1" x2="23" y2="23" />
                  </svg>
                ) : (
                  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                    <circle cx="12" cy="12" r="3" />
                  </svg>
                )}
              </button>
            </div>
          </div>
          {error && (
            <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
          )}
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-accent hover:bg-accent-hover text-stone-900 font-medium py-2.5 transition-colors disabled:opacity-50"
          >
            {loading ? "Signing in…" : "Sign in"}
          </button>
          <div className="mt-4 pt-4 border-t border-stone-200 dark:border-stone-600">
            <button
              type="button"
              onClick={handleViewAsGuest}
              disabled={guestLoading || loading}
              className="w-full rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 text-stone-700 dark:text-stone-300 font-medium py-2.5 transition-colors hover:bg-stone-50 dark:hover:bg-stone-600 disabled:opacity-50"
            >
              {guestLoading ? "Entering…" : "View as guest"}
            </button>
            <p className="text-xs text-stone-500 dark:text-stone-400 mt-2 text-center">
              Same privileges as a guest: only demo books are visible.
            </p>
          </div>
        </form>
      </div>
    </div>
  );
}
