import { useEffect, useState } from "react";
import { adminApi, type AdminUser } from "../../api";
import { PageHeader, Modal, Field, inputClass, Btn } from "../../components/ui";
import { useStore } from "../../store";

type Draft = Partial<AdminUser> & { password?: string };

const EMPTY: Draft = { name: "", email: "", type: "b2c", role: "customer", is_active: true, password: "" };

export function Users() {
  const { user: me } = useStore();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [search, setSearch] = useState("");
  const [editing, setEditing] = useState<Draft | null>(null);
  const [error, setError] = useState("");

  function load() {
    adminApi.users({ search }).then((r) => setUsers(r.data)).catch(() => {});
  }
  useEffect(() => { const t = setTimeout(load, 200); return () => clearTimeout(t); }, [search]);

  async function save() {
    if (!editing) return;
    setError("");
    const payload: Record<string, unknown> = {
      name: editing.name, email: editing.email, type: editing.type,
      role: editing.role, is_active: !!editing.is_active,
    };
    if (editing.password) payload.password = editing.password;
    try {
      await adminApi.saveUser(editing.id ?? null, payload);
      setEditing(null);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    }
  }

  async function remove(u: AdminUser) {
    if (!confirm(`Delete ${u.name}?`)) return;
    try {
      await adminApi.deleteUser(u.id);
      load();
    } catch (e) {
      alert(e instanceof Error ? e.message : "Delete failed");
    }
  }

  return (
    <div>
      <PageHeader title="Users" action={<Btn onClick={() => setEditing({ ...EMPTY })}>+ New user</Btn>} />

      <input placeholder="Search name/email…" value={search} onChange={(e) => setSearch(e.target.value)} className={inputClass + " max-w-xs mb-4"} />

      <div className="card overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-slate-500 text-left">
            <tr>
              <th className="px-4 py-2">Name</th>
              <th className="px-4 py-2">Email</th>
              <th className="px-4 py-2">Type</th>
              <th className="px-4 py-2">Role</th>
              <th className="px-4 py-2">Active</th>
              <th className="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id} className="border-t border-slate-100">
                <td className="px-4 py-2 font-medium">{u.name}</td>
                <td className="px-4 py-2 text-slate-500">{u.email}</td>
                <td className="px-4 py-2 uppercase text-xs">{u.type}</td>
                <td className="px-4 py-2">{u.role}</td>
                <td className="px-4 py-2">{u.is_active ? "✓" : "✕"}</td>
                <td className="px-4 py-2 text-right space-x-1 whitespace-nowrap">
                  <Btn variant="ghost" onClick={() => setEditing({ ...u, password: "" })}>Edit</Btn>
                  {u.id !== me?.id && <Btn variant="danger" onClick={() => remove(u)}>Delete</Btn>}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {editing && (
        <Modal title={editing.id ? "Edit user" : "New user"} onClose={() => setEditing(null)}>
          <Field label="Name">
            <input className={inputClass} value={editing.name ?? ""} onChange={(e) => setEditing({ ...editing, name: e.target.value })} />
          </Field>
          <Field label="Email">
            <input className={inputClass} value={editing.email ?? ""} onChange={(e) => setEditing({ ...editing, email: e.target.value })} />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Type">
              <select className={inputClass} value={editing.type} onChange={(e) => setEditing({ ...editing, type: e.target.value as AdminUser["type"] })}>
                <option value="b2c">b2c</option>
                <option value="b2b">b2b</option>
              </select>
            </Field>
            <Field label="Role">
              <select className={inputClass} value={editing.role} onChange={(e) => setEditing({ ...editing, role: e.target.value })}>
                <option value="customer">customer</option>
                <option value="manager">manager</option>
                <option value="admin">admin</option>
              </select>
            </Field>
          </div>
          <Field label={editing.id ? "New password (leave blank to keep)" : "Password"}>
            <input type="password" className={inputClass} value={editing.password ?? ""} onChange={(e) => setEditing({ ...editing, password: e.target.value })} />
          </Field>
          <label className="flex items-center gap-2 text-sm mb-4">
            <input type="checkbox" checked={!!editing.is_active} onChange={(e) => setEditing({ ...editing, is_active: e.target.checked })} />
            Active
          </label>
          {error && <p className="text-sm text-red-600 mb-3">{error}</p>}
          <div className="flex justify-end gap-2">
            <Btn variant="ghost" onClick={() => setEditing(null)}>Cancel</Btn>
            <Btn onClick={save}>Save</Btn>
          </div>
        </Modal>
      )}
    </div>
  );
}
