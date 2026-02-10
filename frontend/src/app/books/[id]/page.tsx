"use client";

import { useEffect, useState } from "react";
import { useRouter, useParams } from "next/navigation";
import Link from "next/link";
import { fetchBook, getDownloadUrl, deleteBook, isAuthenticated } from "@/lib/api";

export default function BookDetailPage() {
  const router = useRouter();
  const params = useParams();
  const id = params?.id as string;
  const [book, setBook] = useState<Awaited<ReturnType<typeof fetchBook>> | null>(null);
  const [loading, setLoading] = useState(true);
  const [downloading, setDownloading] = useState(false);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
      return;
    }
    if (!id) return;
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
            {book.coverUrl ? (
              <img
                src={book.coverUrl}
                alt=""
                className="w-40 h-60 rounded-lg object-cover shrink-0 mx-auto sm:mx-0 bg-accent-muted/30 dark:bg-accent-muted/20 ring-2 ring-accent/30"
              />
            ) : (
              <div className="w-40 h-60 rounded-lg bg-accent-muted/30 dark:bg-accent-muted/20 shrink-0 mx-auto sm:mx-0 flex items-center justify-center text-accent-muted font-semibold ring-2 ring-accent/30">
                {book.format.toUpperCase()}
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
                <dt className="text-accent-muted dark:text-accent-muted font-medium">Format</dt>
                <dd className="text-stone-900 dark:text-stone-100 uppercase">{book.format}</dd>
                <dt className="text-accent-muted dark:text-accent-muted font-medium">File</dt>
                <dd className="text-stone-900 dark:text-stone-100 truncate">{book.originalName}</dd>
              </dl>
              <div className="mt-6 flex flex-wrap gap-3">
                <button
                  onClick={handleDownload}
                  disabled={downloading}
                  className="rounded-lg bg-accent hover:bg-accent-hover text-stone-900 font-medium px-4 py-2 disabled:opacity-50"
                >
                  {downloading ? "Preparing…" : "Download book"}
                </button>
                <button
                  onClick={handleDelete}
                  disabled={deleting}
                  className="rounded-lg border border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 font-medium px-4 py-2 hover:bg-red-50 dark:hover:bg-red-950/30 disabled:opacity-50"
                >
                  {deleting ? "Deleting…" : "Delete book"}
                </button>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
