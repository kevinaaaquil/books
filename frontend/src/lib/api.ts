/**
 * API base URL (no trailing slash). Set NEXT_PUBLIC_API_BASE_URL in frontend .env or at build time.
 * All backend routes use this base: /api/auth/login, /api/books, /api/books/:id, /api/books/:id/download, /api/upload.
 */
function getApiBaseUrl(): string {
  const url =
    process.env.NEXT_PUBLIC_API_BASE_URL ||
    process.env.NEXT_PUBLIC_API_URL ||
    "http://localhost:8080";
  return url.replace(/\/$/, "");
}

export type Book = {
  id: string;
  title: string;
  authors?: string[];
  publisher?: string;
  publishDate?: string;
  isbn?: string;
  pageCount?: number;
  coverUrl?: string;
  edition?: string;
  format: string;
  originalName: string;
  createdAt: string;
};

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("token");
}

export function setToken(token: string) {
  localStorage.setItem("token", token);
}

export function clearToken() {
  localStorage.removeItem("token");
}

export function isAuthenticated(): boolean {
  return !!getToken();
}

async function authFetch(path: string, options: RequestInit = {}) {
  const token = getToken();
  const headers: HeadersInit = {
    ...(options.headers as Record<string, string>),
  };
  if (token) headers["Authorization"] = `Bearer ${token}`;
  const res = await fetch(`${getApiBaseUrl()}${path}`, { ...options, headers });
  if (res.status === 401) {
    clearToken();
    if (typeof window !== "undefined") window.location.href = "/login";
  }
  return res;
}

export async function login(email: string, password: string): Promise<{ token: string; email: string }> {
  const res = await fetch(`${getApiBaseUrl()}/api/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || "Login failed");
  return data;
}

export async function fetchBooks(): Promise<Book[]> {
  const res = await authFetch("/api/books");
  if (!res.ok) throw new Error("Failed to load books");
  return res.json();
}

export async function fetchBook(id: string): Promise<Book> {
  const res = await authFetch(`/api/books/${id}`);
  if (!res.ok) throw new Error("Book not found");
  return res.json();
}

export async function getDownloadUrl(id: string): Promise<string> {
  const res = await authFetch(`/api/books/${id}/download`);
  if (!res.ok) throw new Error("Failed to get download link");
  const data = await res.json();
  return data.url;
}

export async function deleteBook(id: string): Promise<void> {
  const res = await authFetch(`/api/books/${id}`, { method: "DELETE" });
  if (!res.ok) {
    const text = await res.text();
    try {
      const data = JSON.parse(text) as { error?: string };
      throw new Error(data.error || "Failed to delete book");
    } catch (e) {
      if (e instanceof Error && e.message !== "Failed to delete book") throw e;
      throw new Error("Failed to delete book");
    }
  }
}

export async function uploadBook(file: File): Promise<{ id: string; title: string }> {
  const token = getToken();
  if (!token) throw new Error("Not logged in");
  const form = new FormData();
  form.append("file", file);
  const res = await fetch(`${getApiBaseUrl()}/api/upload`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
    body: form,
  });
  const text = await res.text();
  if (!res.ok) {
    try {
      const data = JSON.parse(text) as { error?: string };
      throw new Error(data.error || "Upload failed");
    } catch (e) {
      if (e instanceof SyntaxError || (typeof text === "string" && text.trimStart().startsWith("<!DOCTYPE")))
        throw new Error("Server returned an error page. Check that the API URL is correct and the backend is running.");
      throw e;
    }
  }
  try {
    return JSON.parse(text) as { id: string; title: string };
  } catch {
    throw new Error("Invalid response from server");
  }
}
