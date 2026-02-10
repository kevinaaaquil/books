"use client";

import { useEffect, useState, useRef } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { fetchBooks, uploadBook, deleteBook, clearToken, isAuthenticated, type Book } from "@/lib/api";

export default function BooksPage() {
  const router = useRouter();
  const [books, setBooks] = useState<Book[]>([]);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState("");
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
      return;
    }
    fetchBooks()
      .then((list) => setBooks(Array.isArray(list) ? list : []))
      .catch(() => setBooks([]))
      .finally(() => setLoading(false));
  }, [router]);

  async function handleUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploadError("");
    setUploading(true);
    try {
      await uploadBook(file);
      const list = await fetchBooks();
      setBooks(Array.isArray(list) ? list : []);
      if (fileInputRef.current) fileInputRef.current.value = "";
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
        <div className="max-w-4xl mx-auto px-4 py-4 flex items-center justify-between">
          <h1 className="text-xl font-semibold text-stone-900 dark:text-stone-100">
            <span className="text-accent">My Books</span>
          </h1>
          <button
            onClick={handleLogout}
            className="text-sm text-accent-muted hover:text-accent font-medium transition-colors"
          >
            Log out
          </button>
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-4 py-8">
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
              <li key={book.id}>
                <Link
                  href={`/books/${book.id}`}
                  className="block rounded-xl border-l-4 border-l-accent border border-stone-200 dark:border-stone-700 bg-white dark:bg-stone-800 p-4 shadow-sm hover:shadow-md hover:border-accent/50 transition-all"
                >
                  <div className="flex gap-4">
                    {book.coverUrl ? (
                      <img
                        src={book.coverUrl}
                        alt=""
                        className="h-24 w-16 rounded object-cover shrink-0 bg-accent-muted/30 dark:bg-accent-muted/20 ring-1 ring-accent/20"
                      />
                    ) : (
                      <div className="h-24 w-16 rounded bg-accent-muted/30 dark:bg-accent-muted/20 shrink-0 flex items-center justify-center text-accent-muted font-semibold text-xs uppercase ring-1 ring-accent/20">
                        {book.format}
                      </div>
                    )}
                    <div className="min-w-0 flex-1">
                      <h2 className="font-medium text-stone-900 dark:text-stone-100 break-words">
                        {book.title}
                      </h2>
                      {book.authors?.length ? (
                        <p className="text-sm text-accent-muted dark:text-accent-muted break-words line-clamp-2">
                          {book.authors.join(", ")}
                        </p>
                      ) : null}
                      <div className="mt-2 flex flex-wrap items-center gap-2">
                        <span className="inline-block text-xs font-medium uppercase px-2 py-0.5 rounded bg-accent/20 text-accent-muted dark:text-accent-muted">
                          {book.format}
                        </span>
                        <button
                          type="button"
                          onClick={(e) => handleDelete(e, book.id)}
                          disabled={deletingId === book.id}
                          className="text-xs font-medium text-red-600 dark:text-red-400 hover:underline disabled:opacity-50"
                          aria-label={`Delete ${book.title}`}
                        >
                          {deletingId === book.id ? "Deleting…" : "Delete"}
                        </button>
                      </div>
                    </div>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </main>
    </div>
  );
}
