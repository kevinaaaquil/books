"use client";

import { useEffect, useState, useRef } from "react";
import { useRouter, useParams } from "next/navigation";
import Link from "next/link";
import { fetchBook, getDownloadUrl, deleteBook, refreshBookMetadata, patchBookViewByGuest, sendToKindle, isAuthenticated, getMe, updateMePreferences, getDisplayCoverUrl, isAdmin, type User } from "@/lib/api";

export default function BookDetailPage() {
  const router = useRouter();
  const params = useParams();
  const id = params?.id as string;
  const [me, setMe] = useState<User | null>(null);
  const [book, setBook] = useState<Awaited<ReturnType<typeof fetchBook>> | null>(null);
  const [loading, setLoading] = useState(true);
  const [downloading, setDownloading] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [refreshError, setRefreshError] = useState("");
  const [refreshIsbn, setRefreshIsbn] = useState("");
  const [showOverwriteWarning, setShowOverwriteWarning] = useState(false);
  const [thumbnailFailed, setThumbnailFailed] = useState(false);
  const [useExtractedCover, setUseExtractedCover] = useState(false);
  const [viewByGuestToggling, setViewByGuestToggling] = useState(false);
  const [optionsOpen, setOptionsOpen] = useState(false);
  const optionsRef = useRef<HTMLDivElement>(null);
  const [sendingToKindle, setSendingToKindle] = useState(false);
  const [sendToKindleError, setSendToKindleError] = useState("");
  const [showKindleSetupModal, setShowKindleSetupModal] = useState(false);
  const [sentToKindleToast, setSentToKindleToast] = useState<string | null>(null);

  const canDelete = me?.role === "admin";
  const canRefresh = me?.role === "admin" || me?.role === "editor";
  const canToggleViewByGuest = isAdmin();

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (optionsRef.current && !optionsRef.current.contains(e.target as Node)) {
        setOptionsOpen(false);
      }
    }
    if (optionsOpen) {
      document.addEventListener("mousedown", handleClickOutside);
      return () => document.removeEventListener("mousedown", handleClickOutside);
    }
  }, [optionsOpen]);

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
      return;
    }
    if (!id) return;
    setThumbnailFailed(false);
    Promise.all([getMe(), fetchBook(id)])
      .then(([user, b]) => {
        setMe(user);
        setUseExtractedCover(user.useExtractedCover ?? false);
        setBook(b);
      })
      .catch(() => setBook(null))
      .finally(() => setLoading(false));
  }, [id, router]);

  async function handleThumbnailToggle() {
    const next = !useExtractedCover;
    try {
      const me = await updateMePreferences({ useExtractedCover: next });
      setUseExtractedCover(me.useExtractedCover ?? next);
    } catch {
      // keep current state on error
    }
  }

  async function handleDownload() {
    if (!id) return;
    setDownloading(true);
    try {
      const url = await getDownloadUrl(id);
      window.open(url, "_blank");
    } finally {
      setDownloading(false);
    }
  }

  async function handleDelete() {
    if (!id) return;
    if (!confirm("If you delete this book, it will be lost forever. Are you sure?")) return;
    setDeleting(true);
    try {
      await deleteBook(id);
      router.push("/books");
      router.refresh();
    } finally {
      setDeleting(false);
    }
  }

  function normalizeIsbn(isbn: string) {
    return isbn.replace(/-/g, "");
  }

  function handleRefreshClick() {
    const isbnInput = refreshIsbn.trim();
    const normalizedInput = normalizeIsbn(isbnInput);
    if (isbnInput && normalizedInput !== normalizeIsbn(book?.isbn ?? "")) {
      setShowOverwriteWarning(true);
      return;
    }
    doRefreshMetadata(isbnInput || undefined);
  }

  async function handleSendToKindle() {
    if (!id) return;
    setSendToKindleError("");
    setShowKindleSetupModal(false);
    setSendingToKindle(true);
    try {
      const result = await sendToKindle(id);
      setSendToKindleError("");
      setSentToKindleToast(result.kindleMail);
      setTimeout(() => setSentToKindleToast(null), 3000);
    } catch (err) {
      const e = err as Error & { code?: string };
      if (e.code === "KINDLE_CONFIG_REQUIRED") {
        setShowKindleSetupModal(true);
      } else {
        setSendToKindleError(e.message || "Failed to send to Kindle");
      }
    } finally {
      setSendingToKindle(false);
    }
  }

  async function handleViewByGuestToggle() {
    if (!id || !book) return;
    const next = !book.viewByGuest;
    setViewByGuestToggling(true);
    try {
      const updated = await patchBookViewByGuest(id, next);
      setBook(updated);
    } catch {
      // keep current state on error
    } finally {
      setViewByGuestToggling(false);
    }
  }

  async function doRefreshMetadata(isbn?: string) {
    if (!id) return;
    setShowOverwriteWarning(false);
    setRefreshError("");
    setRefreshing(true);
    try {
      const updated = await refreshBookMetadata(id, isbn);
      setBook(updated);
      setRefreshIsbn("");
    } catch (err) {
      setRefreshError(err instanceof Error ? err.message : "Failed to refresh metadata");
    } finally {
      setRefreshing(false);
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-accent-soft dark:bg-accent-soft">
        <p className="text-accent-muted dark:text-accent-muted">Loading…</p>
      </div>
    );
  }

  if (!book) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center bg-accent-soft dark:bg-accent-soft gap-4">
        <p className="text-accent-muted dark:text-accent-muted">Book not found.</p>
        <Link href="/books" className="text-accent font-medium hover:underline">
          Back to My Books
        </Link>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-accent-soft dark:bg-accent-soft">
      <header className="border-b-2 border-accent/20 bg-white dark:bg-stone-800 shadow-sm">
        <div className="max-w-3xl mx-auto px-4 py-4 flex items-center justify-between gap-4 flex-wrap">
          <Link
            href="/books"
            className="text-sm font-medium text-accent hover:underline"
          >
            ← My Books
          </Link>
          <label className="flex items-center gap-2 cursor-pointer select-none text-sm text-stone-700 dark:text-stone-300">
            <span className="relative inline-block w-10 h-6 rounded-full">
              <input
                type="checkbox"
                checked={useExtractedCover}
                onChange={handleThumbnailToggle}
                className="sr-only peer"
              />
              <span className="absolute inset-0 rounded-full bg-stone-300 dark:bg-stone-600 peer-checked:bg-accent transition-colors" />
              <span className="absolute left-1 top-1 w-4 h-4 rounded-full bg-white shadow transition-transform peer-checked:translate-x-4" />
            </span>
            <span>Extracted thumbnail</span>
          </label>
        </div>
      </header>

      <main className="max-w-3xl mx-auto px-4 py-8">
        <div className="rounded-xl border-2 border-accent/20 bg-white dark:bg-stone-800 shadow-lg shadow-accent/5 overflow-hidden">
          <div className="p-6 sm:p-8 flex flex-col sm:flex-row gap-6">
            {getDisplayCoverUrl(book, useExtractedCover) && !thumbnailFailed ? (
              <img
                src={getDisplayCoverUrl(book, useExtractedCover)!}
                alt=""
                onError={() => setThumbnailFailed(true)}
                className="w-40 h-60 rounded-lg object-cover shrink-0 mx-auto sm:mx-0 bg-accent-muted/30 dark:bg-accent-muted/20 ring-2 ring-accent/30"
              />
            ) : (
              <div className="w-40 h-60 rounded-lg bg-stone-600 dark:bg-stone-500 shrink-0 mx-auto sm:mx-0 flex items-center justify-center text-white font-semibold text-sm ring-2 ring-accent/30 p-3 text-center break-words">
                {book.originalName || book.format.toUpperCase()}
              </div>
            )}
            <div className="flex-1 min-w-0">
              <h1 className="text-2xl font-semibold text-stone-900 dark:text-stone-100 break-words">
                {book.title}
              </h1>
              {book.authors?.length ? (
                <p className="mt-1 text-accent-muted dark:text-accent-muted">
                  {book.authors.join(", ")}
                </p>
              ) : null}
              <dl className="mt-4 space-y-2 text-sm">
                {book.publisher ? (
                  <>
                    <dt className="text-accent-muted dark:text-accent-muted font-medium">Publisher</dt>
                    <dd className="text-stone-900 dark:text-stone-100">{book.publisher}</dd>
                  </>
                ) : null}
                {book.publishDate ? (
                  <>
                    <dt className="text-accent-muted dark:text-accent-muted font-medium">Published</dt>
                    <dd className="text-stone-900 dark:text-stone-100">{book.publishDate}</dd>
                  </>
                ) : null}
                {book.pageCount != null && book.pageCount > 0 ? (
                  <>
                    <dt className="text-accent-muted dark:text-accent-muted font-medium">Pages</dt>
                    <dd className="text-stone-900 dark:text-stone-100">{book.pageCount}</dd>
                  </>
                ) : null}
                {book.isbn ? (
                  <>
                    <dt className="text-accent-muted dark:text-accent-muted font-medium">ISBN</dt>
                    <dd className="text-stone-900 dark:text-stone-100">{book.isbn}</dd>
                  </>
                ) : null}
                {book.edition ? (
                  <>
                    <dt className="text-accent-muted dark:text-accent-muted font-medium">Edition</dt>
                    <dd className="text-stone-900 dark:text-stone-100">{book.edition}</dd>
                  </>
                ) : null}
                {book.category ? (
                  <>
                    <dt className="text-accent-muted dark:text-accent-muted font-medium">Category</dt>
                    <dd className="text-stone-900 dark:text-stone-100">{book.category}</dd>
                  </>
                ) : null}
                {book.ratingCount != null && book.ratingCount > 0 && book.ratingAverage != null ? (
                  <>
                    <dt className="text-accent-muted dark:text-accent-muted font-medium">Rating</dt>
                    <dd className="text-stone-900 dark:text-stone-100">
                      {book.ratingAverage.toFixed(1)} ★ ({book.ratingCount} {book.ratingCount === 1 ? "rating" : "ratings"})
                    </dd>
                  </>
                ) : null}
                <dt className="text-accent-muted dark:text-accent-muted font-medium">Format</dt>
                <dd className="text-stone-900 dark:text-stone-100 uppercase">{book.format}</dd>
                <dt className="text-accent-muted dark:text-accent-muted font-medium">Uploaded by</dt>
                <dd className="text-stone-900 dark:text-stone-100">{book.uploadedByEmail || "—"}</dd>
                <dt className="text-accent-muted dark:text-accent-muted font-medium">File</dt>
                <dd className="text-stone-900 dark:text-stone-100 truncate">{book.originalName}</dd>
              </dl>
              {book.categories && book.categories.length > 0 ? (
                <div className="mt-3 flex flex-wrap gap-2">
                  {book.categories.slice(0, 10).map((cat) => (
                    <span
                      key={cat}
                      className="inline-block text-xs px-2 py-0.5 rounded bg-stone-200 dark:bg-stone-600 text-stone-700 dark:text-stone-300"
                    >
                      {cat}
                    </span>
                  ))}
                </div>
              ) : null}
              {book.preface ? (
                <div className="mt-4 pt-4 border-t border-stone-200 dark:border-stone-600">
                  <h2 className="text-sm font-medium text-accent-muted dark:text-accent-muted mb-2">Description</h2>
                  <p className="text-sm text-stone-700 dark:text-stone-300 whitespace-pre-wrap line-clamp-6">
                    {book.preface}
                  </p>
                </div>
              ) : null}
              <div className="mt-6 flex flex-wrap items-center gap-3">
                <button
                  onClick={handleDownload}
                  disabled={downloading}
                  className="rounded-lg bg-accent hover:bg-accent-hover text-stone-900 font-medium px-4 py-2 disabled:opacity-50"
                >
                  {downloading ? "Preparing…" : "Download book"}
                </button>
                <button
                  onClick={handleSendToKindle}
                  disabled={sendingToKindle}
                  className="rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-4 py-2 text-sm font-medium text-stone-700 dark:text-stone-300 hover:bg-stone-50 dark:hover:bg-stone-600 disabled:opacity-50"
                >
                  {sendingToKindle ? "Sending…" : "Send to Kindle"}
                </button>
                {canDelete && (
                  <button
                    onClick={handleDelete}
                    disabled={deleting}
                    className="rounded-lg border border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 font-medium px-4 py-2 hover:bg-red-50 dark:hover:bg-red-950/30 disabled:opacity-50"
                  >
                    {deleting ? "Deleting…" : "Delete book"}
                  </button>
                )}
                {canToggleViewByGuest && (
                  <div className="relative" ref={optionsRef}>
                    <button
                      type="button"
                      onClick={() => setOptionsOpen((o) => !o)}
                      className="rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-3 py-2 text-sm font-medium text-stone-700 dark:text-stone-300 hover:bg-stone-50 dark:hover:bg-stone-600"
                    >
                      Options ▾
                    </button>
                    {optionsOpen && (
                      <div className="absolute left-0 top-full z-10 mt-1 min-w-[200px] rounded-lg border border-stone-200 dark:border-stone-600 bg-white dark:bg-stone-800 shadow-lg py-1">
                        <button
                          type="button"
                          onClick={handleViewByGuestToggle}
                          disabled={viewByGuestToggling}
                          className="w-full flex items-center justify-between gap-3 px-3 py-2 text-left text-sm text-stone-700 dark:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-700 disabled:opacity-50"
                        >
                          <span>View by guest (demo)</span>
                          <span className="relative inline-block w-9 h-5 shrink-0 rounded-full">
                            <span
                              className={`absolute inset-0 rounded-full transition-colors ${
                                book.viewByGuest ? "bg-accent" : "bg-stone-300 dark:bg-stone-600"
                              }`}
                            />
                            <span
                              className={`absolute top-0.5 w-4 h-4 rounded-full bg-white shadow transition-transform ${
                                book.viewByGuest ? "left-4" : "left-0.5"
                              }`}
                            />
                          </span>
                        </button>
                      </div>
                    )}
                  </div>
                )}
              </div>
              {sendToKindleError && (
                <p className="mt-2 text-sm text-red-600 dark:text-red-400">{sendToKindleError}</p>
              )}

              {canRefresh && (
                <div className="mt-6 pt-6 border-t border-stone-200 dark:border-stone-600">
                  <h2 className="text-sm font-medium text-stone-700 dark:text-stone-300 mb-2">Refresh metadata</h2>
                  <p className="text-xs text-stone-500 dark:text-stone-400 mb-2">
                    Refetch metadata from the catalog using the book&apos;s ISBN, or enter a different ISBN to overwrite and refetch.
                  </p>
                  <div className="flex flex-wrap items-center gap-2">
                    <input
                      type="text"
                      value={refreshIsbn}
                      onChange={(e) => setRefreshIsbn(e.target.value)}
                      placeholder={book.isbn || "Enter ISBN"}
                      className="rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-3 py-2 text-sm text-stone-900 dark:text-stone-100 w-40"
                    />
                    <a
                      href={`https://www.google.com/search?q=${encodeURIComponent((book?.title ?? "") + " epub isbn")}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="rounded-lg border border-stone-300 dark:border-stone-600 bg-stone-100 dark:bg-stone-600 px-2.5 py-2 text-xs text-stone-600 dark:text-stone-300 hover:bg-stone-200 dark:hover:bg-stone-500 shrink-0"
                      title={`Search "${book?.title ?? ""} ISBN"`}
                    >
                      Search
                    </a>
                    <button
                      onClick={handleRefreshClick}
                      disabled={refreshing || (!refreshIsbn.trim() && !book.isbn)}
                      className="rounded-lg border border-accent/50 text-accent font-medium px-4 py-2 hover:bg-accent/10 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {refreshing ? "Refreshing…" : "Refresh metadata"}
                    </button>
                  </div>
                  {refreshError && <p className="mt-2 text-sm text-red-600 dark:text-red-400">{refreshError}</p>}
                </div>
              )}
            </div>
          </div>
        </div>
      </main>

      {sentToKindleToast && (
        <div
          className="fixed top-6 right-6 z-30 rounded-lg bg-stone-800 dark:bg-stone-700 text-white px-4 py-3 shadow-lg border border-stone-600 dark:border-stone-500 text-sm font-medium"
          role="status"
          aria-live="polite"
        >
          Sent to {sentToKindleToast}
        </div>
      )}

      {showKindleSetupModal && (
        <div
          className="fixed inset-0 z-20 flex items-center justify-center p-4 bg-stone-900/50"
          onClick={() => setShowKindleSetupModal(false)}
        >
          <div
            className="bg-white dark:bg-stone-800 rounded-xl shadow-xl border border-stone-200 dark:border-stone-700 w-full max-w-sm p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="text-lg font-semibold text-stone-900 dark:text-stone-100 mb-2">Set up Kindle config</h2>
            <p className="text-sm text-stone-600 dark:text-stone-400 mb-6">
              Set up your Kindle config to send books to your device. Add your iCloud and Kindle email in Kindle setup.
            </p>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={() => setShowKindleSetupModal(false)}
                className="flex-1 rounded-lg border border-stone-300 dark:border-stone-600 px-4 py-2 text-stone-700 dark:text-stone-300 font-medium"
              >
                Cancel
              </button>
              <Link
                href="/kindle-setup"
                className="flex-1 rounded-lg bg-accent hover:bg-accent-hover text-stone-900 font-medium py-2 px-4 text-center inline-block"
              >
                Setup Kindle
              </Link>
            </div>
          </div>
        </div>
      )}

      {showOverwriteWarning && (
        <div
          className="fixed inset-0 z-20 flex items-center justify-center p-4 bg-stone-900/50"
          onClick={() => setShowOverwriteWarning(false)}
        >
          <div
            className="bg-white dark:bg-stone-800 rounded-xl shadow-xl border border-stone-200 dark:border-stone-700 w-full max-w-sm p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="text-lg font-semibold text-stone-900 dark:text-stone-100 mb-2">Overwrite ISBN?</h2>
            <p className="text-sm text-stone-600 dark:text-stone-400 mb-4">
              Using a new ISBN will overwrite the original ISBN and refetch metadata. This cannot be undone. Continue?
            </p>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => doRefreshMetadata(refreshIsbn.trim() || undefined)}
                disabled={refreshing}
                className="flex-1 rounded-lg bg-accent hover:bg-accent-hover text-stone-900 font-medium py-2 disabled:opacity-50"
              >
                {refreshing ? "Refreshing…" : "Continue"}
              </button>
              <button
                type="button"
                onClick={() => setShowOverwriteWarning(false)}
                className="rounded-lg border border-stone-300 dark:border-stone-600 px-4 py-2 text-stone-700 dark:text-stone-300 font-medium"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
