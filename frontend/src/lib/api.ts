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

/** Returns absolute URL for cover/thumbnail (e.g. /api/books/:id/cover becomes API base + path so auth and cross-origin work). */
export function getBookCoverOrThumbnailUrl(url: string | undefined): string | undefined {
  if (!url) return undefined;
  if (url.startsWith("/")) return getApiBaseUrl() + url;
  return url;
}

/** Display URL for a book's cover: uses extracted cover when preference is true and book has one, else API thumbnail/cover. */
export function getDisplayCoverUrl(
  book: Book,
  useExtractedCover: boolean
): string | undefined {
  const raw =
    useExtractedCover && book.extractedCoverUrl
      ? book.extractedCoverUrl
      : book.thumbnailUrl ?? book.coverUrl;
  return getBookCoverOrThumbnailUrl(raw);
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
  thumbnailUrl?: string;
  edition?: string;
  preface?: string;
  category?: string;
  categories?: string[];
  ratingAverage?: number;
  ratingCount?: number;
  format: string;
  originalName: string;
  uploadedByEmail?: string;
  extractedCoverUrl?: string;
  createdAt: string;
};

export type User = {
  id: string;
  email: string;
  role: string;
  useExtractedCover?: boolean;
  createdAt: string;
};

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("token");
}

const ROLE_KEY = "role";

export function setToken(token: string) {
  localStorage.setItem("token", token);
}

export function setRole(role: string) {
  localStorage.setItem(ROLE_KEY, role);
}

export function getRole(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(ROLE_KEY);
}

export function clearToken() {
  localStorage.removeItem("token");
  localStorage.removeItem(ROLE_KEY);
}

export function isAuthenticated(): boolean {
  return !!getToken();
}

export function isAdmin(): boolean {
  return getRole() === "admin";
}

/** Can list, view, and download books. */
export function canReadBooks(): boolean {
  const r = getRole();
  return r === "admin" || r === "editor" || r === "viewer";
}

/** Can upload books. */
export function canUploadBooks(): boolean {
  const r = getRole();
  return r === "admin" || r === "editor";
}

/** Can delete books. */
export function canDeleteBooks(): boolean {
  return getRole() === "admin";
}

/** Can refresh book metadata (editor, admin). */
export function canRefreshMetadata(): boolean {
  const r = getRole();
  return r === "admin" || r === "editor";
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

export async function login(email: string, password: string): Promise<{ token: string; email: string; role?: string }> {
  const res = await fetch(`${getApiBaseUrl()}/api/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || "Login failed");
  return data;
}

export async function getMe(): Promise<User> {
  const res = await authFetch("/api/me");
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error((data as { error?: string }).error || "Failed to load profile");
  const me = data as User;
  if (me.role != null) setRole(me.role);
  return me;
}

export async function updateMePreferences(prefs: { useExtractedCover: boolean }): Promise<User> {
  const res = await authFetch("/api/me/preferences", {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(prefs),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error((data as { error?: string }).error || "Failed to update preference");
  return data as User;
}

export const USER_ROLES = ["viewer", "editor"] as const;
export type CreateUserRole = (typeof USER_ROLES)[number];

export async function createUser(email: string, password: string, role: string): Promise<{ id: string; email: string; role: string }> {
  const res = await authFetch("/api/users", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password, role }),
  });
  const text = await res.text();
  if (!res.ok) {
    try {
      const data = JSON.parse(text) as { error?: string };
      throw new Error(data.error || "Failed to create user");
    } catch (e) {
      if (e instanceof Error) throw e;
      throw new Error("Failed to create user");
    }
  }
  return JSON.parse(text) as { id: string; email: string; role: string };
}

export async function listUsers(): Promise<User[]> {
  const res = await authFetch("/api/users");
  if (!res.ok) throw new Error("Failed to load users");
  return res.json();
}

export async function updateUser(
  id: string,
  body: { email?: string; password?: string; role?: string }
): Promise<User> {
  const res = await authFetch(`/api/users/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const text = await res.text();
  if (!res.ok) {
    try {
      const data = JSON.parse(text) as { error?: string };
      throw new Error(data.error || "Failed to update user");
    } catch (e) {
      if (e instanceof Error) throw e;
      throw new Error("Failed to update user");
    }
  }
  return JSON.parse(text) as User;
}

export async function deleteUser(id: string): Promise<void> {
  const res = await authFetch(`/api/users/${id}`, { method: "DELETE" });
  if (!res.ok) {
    const text = await res.text();
    try {
      const data = JSON.parse(text) as { error?: string };
      throw new Error(data.error || "Failed to delete user");
    } catch (e) {
      if (e instanceof Error) throw e;
      throw new Error("Failed to delete user");
    }
  }
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

/** Refetch metadata by ISBN and update the book. Pass isbn to use a new ISBN (overwrites existing); omit to use book's current ISBN. */
export async function refreshBookMetadata(id: string, isbn?: string): Promise<Book> {
  const res = await authFetch(`/api/books/${id}/refresh-metadata`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(isbn != null && isbn !== "" ? { isbn: isbn.trim().replace(/-/g, "") } : {}),
  });
  const text = await res.text();
  if (!res.ok) {
    try {
      const data = JSON.parse(text) as { error?: string };
      throw new Error(data.error || "Failed to refresh metadata");
    } catch (e) {
      if (e instanceof Error) throw e;
      throw new Error("Failed to refresh metadata");
    }
  }
  return JSON.parse(text) as Book;
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

export async function uploadBook(file: File): Promise<{ id: string; title: string; noISBNFound?: boolean }> {
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
    return JSON.parse(text) as { id: string; title: string; noISBNFound?: boolean };
  } catch {
    throw new Error("Invalid response from server");
  }
}
