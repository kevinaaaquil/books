"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import {
  isAuthenticated,
  getMe,
  setRole,
  getEmailConfig,
  saveEmailConfig,
  clearToken,
  type EmailConfig,
} from "@/lib/api";
import { ProfileMenu } from "@/components/ProfileMenu";

const emptyConfig: EmailConfig = {
  appSpecificPassword: "",
  icloudMail: "",
  senderMail: "",
  kindleMail: "",
};

export default function EmailSetupPage() {
  const router = useRouter();
  const [me, setMe] = useState<{ email?: string; role?: string } | null>(null);
  const [loading, setLoading] = useState(true);
  const [config, setConfig] = useState<EmailConfig>(emptyConfig);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);
  const [isGuest, setIsGuest] = useState(false);

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
      return;
    }
    getMe()
      .then((user) => {
        setMe(user);
        if (user.role != null) setRole(user.role);
        setIsGuest(user.role === "guest");
        return getEmailConfig();
      })
      .then((cfg) => {
        if (cfg) setConfig(cfg);
      })
      .catch(() => setConfig(emptyConfig))
      .finally(() => setLoading(false));
  }, [router]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setSuccess(false);
    setSaving(true);
    try {
      await saveEmailConfig(config);
      setSuccess(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save");
    } finally {
      setSaving(false);
    }
  }

  function handleLogout() {
    clearToken();
    router.replace("/login");
    router.refresh();
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-accent-soft dark:bg-accent-soft">
        <p className="text-accent-muted dark:text-accent-muted">Loading…</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-accent-soft dark:bg-accent-soft">
      <header className="border-b-2 border-accent/20 bg-white dark:bg-stone-800 shadow-sm">
        <div className="max-w-2xl mx-auto px-4 py-4 flex items-center justify-between">
          <h1 className="text-xl font-semibold text-stone-900 dark:text-stone-100">
            <span className="text-accent">Kindle setup</span>
          </h1>
          <div className="flex items-center gap-3">
            <Link
              href="/books"
              className="text-sm font-medium text-accent-muted hover:text-accent transition-colors"
            >
              ← Books
            </Link>
            <ProfileMenu email={me?.email ?? ""} onLogout={handleLogout} />
          </div>
        </div>
      </header>

      <main className="max-w-2xl mx-auto px-4 py-8">
        <p className="text-sm text-stone-600 dark:text-stone-400 mb-6">
          Optional Feature. Configure iCloud and Kindle email so you can send books to your Kindle. The password is stored securely in the database.
        </p>
        {isGuest && (
          <p className="text-sm text-amber-600 dark:text-amber-400 mb-4">
            Guests cannot change email settings. Please contact the administrator to change your role.
          </p>
        )}
        <form
          onSubmit={handleSubmit}
          className="rounded-xl border-2 border-accent/20 bg-white dark:bg-stone-800 p-6 space-y-4"
        >
          <div>
            <label htmlFor="icloudMail" className="block text-sm font-medium text-accent-muted mb-1">
              iCloud Mail
            </label>
            <input
              id="icloudMail"
              type="email"
              value={config.icloudMail}
              onChange={(e) => setConfig((c) => ({ ...c, icloudMail: e.target.value }))}
              readOnly={isGuest}
              className="w-full rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-3 py-2 text-stone-900 dark:text-stone-100 focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-70 disabled:cursor-not-allowed"
              placeholder="you@icloud.com"
            />
          </div>
          <div>
            <label htmlFor="appSpecificPassword" className="block text-sm font-medium text-accent-muted mb-1">
              App-specific password
            </label>
            <input
              id="appSpecificPassword"
              type="password"
              value={config.appSpecificPassword}
              onChange={(e) => setConfig((c) => ({ ...c, appSpecificPassword: e.target.value }))}
              readOnly={isGuest}
              className="w-full rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-3 py-2 text-stone-900 dark:text-stone-100 focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-70 disabled:cursor-not-allowed"
              placeholder="••••••••"
              autoComplete="off"
            />
            <p className="mt-1 text-xs text-stone-500 dark:text-stone-400">
              Generate at appleid.apple.com → Sign-In and Security → App-Specific Passwords.
            </p>
          </div>
          <div>
            <label htmlFor="senderMail" className="block text-sm font-medium text-accent-muted mb-1">
              Sender mail
            </label>
            <input
              id="senderMail"
              type="email"
              value={config.senderMail}
              onChange={(e) => setConfig((c) => ({ ...c, senderMail: e.target.value }))}
              readOnly={isGuest}
              className="w-full rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-3 py-2 text-stone-900 dark:text-stone-100 focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-70 disabled:cursor-not-allowed"
              placeholder="sender@example.com"
            />
          </div>
          <div>
            <label htmlFor="kindleMail" className="block text-sm font-medium text-accent-muted mb-1">
              Kindle mail
            </label>
            <input
              id="kindleMail"
              type="email"
              value={config.kindleMail}
              onChange={(e) => setConfig((c) => ({ ...c, kindleMail: e.target.value }))}
              readOnly={isGuest}
              className="w-full rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-3 py-2 text-stone-900 dark:text-stone-100 focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-70 disabled:cursor-not-allowed"
              placeholder="yourkindle@kindle.com"
            />
          </div>
          {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
          {success && (
            <p className="text-sm text-green-600 dark:text-green-400">Kindle config saved.</p>
          )}
          <div className="flex gap-3 pt-2">
            <button
              type="submit"
              disabled={saving || isGuest}
              className="rounded-lg bg-accent hover:bg-accent-hover text-stone-900 font-medium px-4 py-2 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {saving ? "Saving…" : "Save"}
            </button>
            <Link
              href="/books"
              className="rounded-lg border border-stone-300 dark:border-stone-600 px-4 py-2 text-stone-700 dark:text-stone-300 font-medium inline-block"
            >
              Cancel
            </Link>
          </div>
        </form>
      </main>
    </div>
  );
}
