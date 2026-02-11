"use client";

import { useEffect, useState, useRef } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { fetchBooks, uploadBook, deleteBook, clearToken, isAuthenticated, isAdmin, getMe, updateMePreferences, getDisplayCoverUrl, type Book, type User } from "@/lib/api";

export default function BooksPage() {
  const router = useRouter();
  const [me, setMe] = useState<User | null>(null);
  const [books, setBooks] = useState<Book[]>([]);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState("");
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [noISBNNotification, setNoISBNNotification] = useState<string | null>(null);
  const [failedThumbnailIds, setFailedThumbnailIds] = useState<Set<string>>(new Set());
  const [useExtractedCover, setUseExtractedCover] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const canUpload = me?.role === "admin" || me?.role === "editor";
  const canDelete = me?.role === "admin";

  function handleThumbnailError(bookId: string) {
    setFailedThumbnailIds((prev) => new Set(prev).add(bookId));
  }

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
      return;
    }
    Promise.all([getMe(), fetchBooks()])
      .then(([user, list]) => {
        setMe(user);
        setUseExtractedCover(user.useExtractedCover ?? false);
        setBooks(Array.isArray(list) ? list : []);
      })
      .catch(() => setBooks([]))
      .finally(() => setLoading(false));
  }, [router]);

  async function handleUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploadError("");
    setNoISBNNotification(null);
    setUploading(true);
    try {
      const result = await uploadBook(file);
      const list = await fetchBooks();
      setBooks(Array.isArray(list) ? list : []);
      if (fileInputRef.current) fileInputRef.current.value = "";
      if (result.noISBNFound) {
        setNoISBNNotification("No ISBN was found in this EPUB. The book was uploaded but metadata was not fetched.");
      }
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setUploading(false);
    }
  }

  async function handleDelete(e: React.MouseEvent, id: string) {
    e.preventDefault();
    e.stopPropagation();
    if (!confirm("If you delete this book, it will be lost forever. Are you sure?")) return;
    setDeletingId(id);
    try {
      await deleteBook(id);
      setBooks((prev) => prev.filter((b) => b.id !== id));
    } catch {
      // keep list as is on error
    } finally {
      setDeletingId(null);
    }
  }

  async function handleThumbnailToggle() {
    const next = !useExtractedCover;
    try {
      const me = await updateMePreferences({ useExtractedCover: next });
      setUseExtractedCover(me.useExtractedCover ?? next);
    } catch {
      // keep current state on error
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

  const bookList = Array.isArray(books) ? books : [];

  return (
    <div className="min-h-screen bg-accent-soft dark:bg-accent-soft">
      <header className="border-b-2 border-accent/20 bg-white dark:bg-stone-800 shadow-sm">
        <div className="max-w-4xl mx-auto px-4 py-4 flex items-center justify-between gap-4 flex-wrap">
          <h1 className="text-xl font-semibold text-stone-900 dark:text-stone-100">
            <span className="text-accent">My Books</span>
          </h1>
          <div className="flex items-center gap-3 flex-wrap">
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
            {isAdmin() && (
              <Link
                href="/users"
                className="text-sm font-medium px-3 py-1.5 rounded-lg border border-accent/50 text-accent hover:bg-accent/10 transition-colors"
              >
                Manage users
              </Link>
            )}
            <button
              onClick={handleLogout}
              className="text-sm text-accent-muted hover:text-accent font-medium transition-colors"
            >
              Log out
            </button>
          </div>
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-4 py-8">
        {canUpload && (
          <div className="mb-6 flex flex-wrap items-center gap-4 rounded-xl bg-white dark:bg-stone-800 border border-accent/20 p-4">
            <input
              ref={fileInputRef}
              type="file"
              accept=".epub,.pdf"
              onChange={handleUpload}
              className="hidden"
            />
            <button
              onClick={() => fileInputRef.current?.click()}
              disabled={uploading}
              className="rounded-lg bg-accent hover:bg-accent-hover text-stone-900 font-medium px-4 py-2 disabled:opacity-50"
            >
              {uploading ? "Uploading…" : "Upload book (EPUB or PDF)"}
            </button>
            {uploadError && (
              <p className="text-sm text-red-600 dark:text-red-400">{uploadError}</p>
            )}
          </div>
        )}

        {bookList.length === 0 ? (
          <div className="rounded-xl border-2 border-dashed border-accent/40 bg-accent-soft/50 dark:bg-accent-soft/50 p-12 text-center">
            <p className="text-accent-muted dark:text-accent-muted mb-1">
              No books yet.
            </p>
            <p className="text-stone-600 dark:text-stone-400 text-sm">
              Upload an EPUB or PDF to get started.
            </p>
          </div>
        ) : (
          <ul className="grid gap-4 sm:grid-cols-2">
            {bookList.map((book) => (
              <li key={book.id} className="min-h-[132px]">
                <Link
                  href={`/books/${book.id}`}
                  className="block h-full min-h-[132px] rounded-xl border-l-4 border-l-accent border border-stone-200 dark:border-stone-700 bg-white dark:bg-stone-800 p-4 shadow-sm hover:shadow-md hover:border-accent/50 transition-all"
                >
                  <div className="flex gap-4 h-full">
                    {getDisplayCoverUrl(book, useExtractedCover) && !failedThumbnailIds.has(book.id) ? (
                      <img
                        src={getDisplayCoverUrl(book, useExtractedCover)!}
                        alt=""
                        onError={() => handleThumbnailError(book.id)}
                        className="h-24 w-16 rounded object-cover shrink-0 bg-accent-muted/30 dark:bg-accent-muted/20 ring-1 ring-accent/20"
                      />
                    ) : (
                      <div className="h-24 w-16 rounded bg-stone-600 dark:bg-stone-500 shrink-0 flex items-center justify-center text-white font-semibold text-[10px] ring-1 ring-accent/20 p-1 text-center break-all leading-tight">
                        {book.originalName || book.format}
                      </div>
                    )}
                    <div className="min-w-0 flex-1 flex flex-col">
                      <h2
                        className="font-medium text-stone-900 dark:text-stone-100 truncate"
                        title={book.title}
                      >
                        {book.title}
                      </h2>
                      {book.authors?.length ? (
                        <p
                          className="text-sm text-accent-muted dark:text-accent-muted truncate mt-0.5"
                          title={book.authors.join(", ")}
                        >
                          {book.authors.join(", ")}
                        </p>
                      ) : null}
                      {book.uploadedByEmail ? (
                        <p className="mt-1 text-xs text-stone-500 dark:text-stone-400 truncate">
                          Uploaded by {book.uploadedByEmail}
                        </p>
                      ) : null}
                      <div className="mt-auto pt-2 flex flex-wrap items-center gap-2">
                        <span className="inline-block text-xs font-medium uppercase px-2 py-0.5 rounded bg-stone-600 dark:bg-stone-500 text-white">
                          {book.format}
                        </span>
                        {canDelete && (
                          <button
                            type="button"
                            onClick={(e) => handleDelete(e, book.id)}
                            disabled={deletingId === book.id}
                            className="text-xs font-medium text-red-600 dark:text-red-400 hover:underline disabled:opacity-50"
                            aria-label={`Delete ${book.title}`}
                          >
                            {deletingId === book.id ? "Deleting…" : "Delete"}
                          </button>
                        )}
                      </div>
                    </div>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </main>

      {noISBNNotification && (
        <div
          className="fixed inset-0 z-20 flex items-center justify-center p-4 bg-stone-900/50"
          onClick={() => setNoISBNNotification(null)}
        >
          <div
            className="bg-white dark:bg-stone-800 rounded-xl shadow-xl border border-stone-200 dark:border-stone-700 w-full max-w-sm p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="text-lg font-semibold text-stone-900 dark:text-stone-100 mb-2">Notice</h2>
            <p className="text-sm text-stone-600 dark:text-stone-400 mb-4">{noISBNNotification}</p>
            <button
              type="button"
              onClick={() => setNoISBNNotification(null)}
              className="w-full rounded-lg bg-accent hover:bg-accent-hover text-stone-900 font-medium py-2"
            >
              OK
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
