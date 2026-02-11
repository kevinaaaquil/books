"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import {
  isAuthenticated,
  clearToken,
  listUsers,
  createUser,
  updateUser,
  deleteUser,
  getMe,
  setRole,
  USER_ROLES,
  type User,
} from "@/lib/api";

const ALL_ROLES = ["admin", ...USER_ROLES];

export default function ManageUsersPage() {
  const router = useRouter();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [addOpen, setAddOpen] = useState(false);
  const [editUser, setEditUser] = useState<User | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [addEmail, setAddEmail] = useState("");
  const [addPassword, setAddPassword] = useState("");
  const [addRole, setAddRole] = useState("viewer");
  const [addError, setAddError] = useState("");
  const [addSubmitting, setAddSubmitting] = useState(false);
  const [editEmail, setEditEmail] = useState("");
  const [editPassword, setEditPassword] = useState("");
  const [editRole, setEditRole] = useState("viewer");
  const [editError, setEditError] = useState("");
  const [editSubmitting, setEditSubmitting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
      return;
    }
    // Use server role so any admin can see the page (don't rely only on localStorage)
    getMe()
      .then((me) => {
        if (me.role != null) setRole(me.role);
        if (me.role !== "admin") {
          router.replace("/books");
          return;
        }
        return listUsers();
      })
      .then((list) => {
        if (Array.isArray(list)) setUsers(list);
      })
      .catch(() => setUsers([]))
      .finally(() => setLoading(false));
  }, [router]);

  function handleLogout() {
    clearToken();
    router.replace("/login");
    router.refresh();
  }

  async function handleAddUser(e: React.FormEvent) {
    e.preventDefault();
    setAddError("");
    setAddSubmitting(true);
    try {
      await createUser(addEmail.trim(), addPassword, addRole);
      const list = await listUsers();
      setUsers(list);
      setAddOpen(false);
      setAddEmail("");
      setAddPassword("");
      setAddRole("viewer");
    } catch (err) {
      setAddError(err instanceof Error ? err.message : "Failed to create user");
    } finally {
      setAddSubmitting(false);
    }
  }

  function openEdit(u: User) {
    setEditUser(u);
    setEditEmail(u.email);
    setEditPassword("");
    setEditRole(u.role);
    setEditError("");
  }

  async function handleEditUser(e: React.FormEvent) {
    e.preventDefault();
    if (!editUser) return;
    setEditError("");
    setEditSubmitting(true);
    try {
      const body: { email?: string; password?: string; role?: string } = {
        email: editEmail.trim(),
        role: editRole,
      };
      if (editPassword) body.password = editPassword;
      await updateUser(editUser.id, body);
      const list = await listUsers();
      setUsers(list);
      setEditUser(null);
      // Sync role from server so UI updates when current user's role was changed
      getMe()
        .then((m) => {
          if (m.role != null) setRole(m.role);
          if (m.role !== "admin") router.replace("/books");
        })
        .catch(() => {});
    } catch (err) {
      setEditError(err instanceof Error ? err.message : "Failed to update user");
    } finally {
      setEditSubmitting(false);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Delete this user? They will no longer be able to sign in.")) return;
    setDeleteError(null);
    setDeletingId(id);
    try {
      await deleteUser(id);
      setUsers((prev) => prev.filter((u) => u.id !== id));
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : "Failed to delete user");
    } finally {
      setDeletingId(null);
    }
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
        <div className="max-w-4xl mx-auto px-4 py-4 flex items-center justify-between">
          <h1 className="text-xl font-semibold text-stone-900 dark:text-stone-100">
            <span className="text-accent">Manage users</span>
          </h1>
          <div className="flex items-center gap-3">
            <Link
              href="/books"
              className="text-sm font-medium text-accent-muted hover:text-accent transition-colors"
            >
              ← Books
            </Link>
            <button
              type="button"
              onClick={() => setAddOpen(true)}
              className="text-sm font-medium px-3 py-1.5 rounded-lg bg-accent hover:bg-accent-hover text-stone-900"
            >
              Add user
            </button>
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
        <div className="rounded-xl border-2 border-accent/20 bg-white dark:bg-stone-800 overflow-hidden">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-stone-200 dark:border-stone-700 bg-stone-50 dark:bg-stone-800/80">
                <th className="px-4 py-3 font-medium text-stone-700 dark:text-stone-300">Email</th>
                <th className="px-4 py-3 font-medium text-stone-700 dark:text-stone-300">Role</th>
                <th className="px-4 py-3 font-medium text-stone-700 dark:text-stone-300">Created</th>
                <th className="px-4 py-3 font-medium text-stone-700 dark:text-stone-300">Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id} className="border-b border-stone-100 dark:border-stone-700/50">
                  <td className="px-4 py-3 text-stone-900 dark:text-stone-100">{u.email}</td>
                  <td className="px-4 py-3">
                    <span className="inline-block px-2 py-0.5 rounded bg-stone-200 dark:bg-stone-600 text-stone-800 dark:text-stone-200 text-xs font-medium">
                      {u.role}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-stone-600 dark:text-stone-400">
                    {new Date(u.createdAt).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 flex gap-2">
                    <button
                      type="button"
                      onClick={() => openEdit(u)}
                      className="text-accent hover:underline font-medium text-xs"
                    >
                      Edit
                    </button>
                    <button
                      type="button"
                      onClick={() => handleDelete(u.id)}
                      disabled={deletingId === u.id}
                      className="text-red-600 dark:text-red-400 hover:underline font-medium text-xs disabled:opacity-50"
                    >
                      {deletingId === u.id ? "Deleting…" : "Delete"}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {users.length === 0 && (
            <p className="px-4 py-8 text-center text-accent-muted dark:text-accent-muted text-sm">
              No users yet. Add one to get started.
            </p>
          )}
        </div>
      </main>

      {addOpen && (
        <div
          className="fixed inset-0 z-10 flex items-center justify-center p-4 bg-stone-900/50"
          onClick={() => setAddOpen(false)}
        >
          <div
            className="bg-white dark:bg-stone-800 rounded-xl shadow-xl border border-stone-200 dark:border-stone-700 w-full max-w-sm p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="text-lg font-semibold text-stone-900 dark:text-stone-100 mb-4">Add user</h2>
            <form onSubmit={handleAddUser} className="space-y-4">
              <div>
                <label htmlFor="add-email" className="block text-sm font-medium text-accent-muted mb-1">
                  Email
                </label>
                <input
                  id="add-email"
                  type="email"
                  value={addEmail}
                  onChange={(e) => setAddEmail(e.target.value)}
                  required
                  className="w-full rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-3 py-2 text-stone-900 dark:text-stone-100"
                />
              </div>
              <div>
                <label htmlFor="add-password" className="block text-sm font-medium text-accent-muted mb-1">
                  Password
                </label>
                <input
                  id="add-password"
                  type="password"
                  value={addPassword}
                  onChange={(e) => setAddPassword(e.target.value)}
                  required
                  className="w-full rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-3 py-2 text-stone-900 dark:text-stone-100"
                />
              </div>
              <div>
                <label htmlFor="add-role" className="block text-sm font-medium text-accent-muted mb-1">
                  Role
                </label>
                <select
                  id="add-role"
                  value={addRole}
                  onChange={(e) => setAddRole(e.target.value)}
                  className="w-full rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-3 py-2 text-stone-900 dark:text-stone-100"
                >
                  {USER_ROLES.map((r) => (
                    <option key={r} value={r}>
                      {r.replace("_", " ")}
                    </option>
                  ))}
                </select>
              </div>
              {addError && <p className="text-sm text-red-600 dark:text-red-400">{addError}</p>}
              <div className="flex gap-2 pt-2">
                <button
                  type="submit"
                  disabled={addSubmitting}
                  className="flex-1 rounded-lg bg-accent hover:bg-accent-hover text-stone-900 font-medium py-2 disabled:opacity-50"
                >
                  {addSubmitting ? "Creating…" : "Create"}
                </button>
                <button
                  type="button"
                  onClick={() => setAddOpen(false)}
                  className="rounded-lg border border-stone-300 dark:border-stone-600 px-4 py-2 text-stone-700 dark:text-stone-300 font-medium"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {editUser && (
        <div
          className="fixed inset-0 z-10 flex items-center justify-center p-4 bg-stone-900/50"
          onClick={() => setEditUser(null)}
        >
          <div
            className="bg-white dark:bg-stone-800 rounded-xl shadow-xl border border-stone-200 dark:border-stone-700 w-full max-w-sm p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="text-lg font-semibold text-stone-900 dark:text-stone-100 mb-4">Edit user</h2>
            <form onSubmit={handleEditUser} className="space-y-4">
              <div>
                <label htmlFor="edit-email" className="block text-sm font-medium text-accent-muted mb-1">
                  Email
                </label>
                <input
                  id="edit-email"
                  type="email"
                  value={editEmail}
                  onChange={(e) => setEditEmail(e.target.value)}
                  required
                  className="w-full rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-3 py-2 text-stone-900 dark:text-stone-100"
                />
              </div>
              <div>
                <label htmlFor="edit-password" className="block text-sm font-medium text-accent-muted mb-1">
                  New password (leave blank to keep)
                </label>
                <input
                  id="edit-password"
                  type="password"
                  value={editPassword}
                  onChange={(e) => setEditPassword(e.target.value)}
                  className="w-full rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-3 py-2 text-stone-900 dark:text-stone-100"
                />
              </div>
              <div>
                <label htmlFor="edit-role" className="block text-sm font-medium text-accent-muted mb-1">
                  Role
                </label>
                <select
                  id="edit-role"
                  value={editRole}
                  onChange={(e) => setEditRole(e.target.value)}
                  className="w-full rounded-lg border border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-700 px-3 py-2 text-stone-900 dark:text-stone-100"
                >
                  {ALL_ROLES.map((r) => (
                    <option key={r} value={r}>
                      {r.replace("_", " ")}
                    </option>
                  ))}
                </select>
              </div>
              {editError && <p className="text-sm text-red-600 dark:text-red-400">{editError}</p>}
              <div className="flex gap-2 pt-2">
                <button
                  type="submit"
                  disabled={editSubmitting}
                  className="flex-1 rounded-lg bg-accent hover:bg-accent-hover text-stone-900 font-medium py-2 disabled:opacity-50"
                >
                  {editSubmitting ? "Saving…" : "Save"}
                </button>
                <button
                  type="button"
                  onClick={() => setEditUser(null)}
                  className="rounded-lg border border-stone-300 dark:border-stone-600 px-4 py-2 text-stone-700 dark:text-stone-300 font-medium"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {deleteError && (
        <div
          className="fixed inset-0 z-20 flex items-center justify-center p-4 bg-stone-900/50"
          onClick={() => setDeleteError(null)}
        >
          <div
            className="bg-white dark:bg-stone-800 rounded-xl shadow-xl border border-stone-200 dark:border-stone-700 w-full max-w-sm p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="text-lg font-semibold text-stone-900 dark:text-stone-100 mb-2">Cannot delete user</h2>
            <p className="text-sm text-stone-600 dark:text-stone-400 mb-4">{deleteError}</p>
            <button
              type="button"
              onClick={() => setDeleteError(null)}
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
