"use client";

import { useEffect, useState } from "react";
import { useRouter, useParams } from "next/navigation";
import Link from "next/link";
import { fetchBook, getDownloadUrl, deleteBook, refreshBookMetadata, isAuthenticated, canDeleteBooks, canRefreshMetadata, getBookCoverOrThumbnailUrl } from "@/lib/api";

export default function BookDetailPage() {
  const router = useRouter();
  const params = useParams();
  const id = params?.id as string;
  const [book, setBook] = useState<Awaited<ReturnType<typeof fetchBook>> | null>(null);
  const [loading, setLoading] = useState(true);
  const [downloading, setDownloading] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [refreshError, setRefreshError] = useState("");
  const [refreshIsbn, setRefreshIsbn] = useState("");
  const [showOverwriteWarning, setShowOverwriteWarning] = useState(false);
  const [thumbnailFailed, setThumbnailFailed] = useState(false);

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
      return;
    }
    if (!id) return;
    setThumbnailFailed(false);
    fetchBook(id)
      .then(setBook)
      .catch(() => setBook(null))
      .finally(() => setLoading(false));
  }, [id, router]);

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

  function handleRefreshClick() {
    const isbnInput = refreshIsbn.trim();
    if (isbnInput && isbnInput !== book?.isbn) {
      setShowOverwriteWarning(true);
      return;
    }
    doRefreshMetadata(isbnInput || undefined);
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
        <div className="max-w-3xl mx-auto px-4 py-4">
          <Link
            href="/books"
            className="text-sm font-medium text-accent hover:underline"
          >
            ← My Books
          </Link>
        </div>
      </header>

      <main className="max-w-3xl mx-auto px-4 py-8">
        <div className="rounded-xl border-2 border-accent/20 bg-white dark:bg-stone-800 shadow-lg shadow-accent/5 overflow-hidden">
          <div className="p-6 sm:p-8 flex flex-col sm:flex-row gap-6">
            {getBookCoverOrThumbnailUrl(book.thumbnailUrl ?? book.coverUrl) && !thumbnailFailed ? (
              <img
                src={getBookCoverOrThumbnailUrl(book.thumbnailUrl ?? book.coverUrl)!}
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
              <div className="mt-6 flex flex-wrap gap-3">
                <button
                  onClick={handleDownload}
                  disabled={downloading}
                  className="rounded-lg bg-accent hover:bg-accent-hover text-stone-900 font-medium px-4 py-2 disabled:opacity-50"
                >
                  {downloading ? "Preparing…" : "Download book"}
                </button>
                {canDeleteBooks() && (
                  <button
                    onClick={handleDelete}
                    disabled={deleting}
                    className="rounded-lg border border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 font-medium px-4 py-2 hover:bg-red-50 dark:hover:bg-red-950/30 disabled:opacity-50"
                  >
                    {deleting ? "Deleting…" : "Delete book"}
                  </button>
                )}
              </div>

              {canRefreshMetadata() && (
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
                onClick={() => doRefreshMetadata(refreshIsbn.trim())}
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
