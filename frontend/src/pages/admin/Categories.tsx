import { useEffect, useState } from "react";
import { adminApi, type AdminCategory } from "../../api";
import { PageHeader, Modal, Field, inputClass, Btn } from "../../components/ui";

type Draft = Partial<AdminCategory>;
const EMPTY: Draft = { name: "", position: 0, is_active: true, parent_id: null };

export function Categories() {
  const [items, setItems] = useState<AdminCategory[]>([]);
  const [editing, setEditing] = useState<Draft | null>(null);
  const [error, setError] = useState("");

  function load() { adminApi.categories().then(setItems).catch(() => {}); }
  useEffect(() => { load(); }, []);

  async function save() {
    if (!editing) return;
    setError("");
    try {
      await adminApi.saveCategory(editing.id ?? null, {
        name: editing.name,
        parent_id: editing.parent_id || null,
        position: Number(editing.position ?? 0),
        is_active: !!editing.is_active,
      });
      setEditing(null);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    }
  }

  async function remove(c: AdminCategory) {
    if (!confirm(`Delete ${c.name}?`)) return;
    await adminApi.deleteCategory(c.id);
    load();
  }

  return (
    <div>
      <PageHeader title="Categories" action={<Btn onClick={() => setEditing({ ...EMPTY })}>+ New category</Btn>} />

      <div className="card overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-slate-500 text-left">
            <tr>
              <th className="px-4 py-2">Name</th>
              <th className="px-4 py-2">Slug</th>
              <th className="px-4 py-2 text-right">Products</th>
              <th className="px-4 py-2">Active</th>
              <th className="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {items.map((c) => (
              <tr key={c.id} className="border-t border-slate-100">
                <td className="px-4 py-2 font-medium">{c.name}</td>
                <td className="px-4 py-2 text-slate-400 font-mono text-xs">{c.slug}</td>
                <td className="px-4 py-2 text-right">{c.products_count ?? 0}</td>
                <td className="px-4 py-2">{c.is_active ? "✓" : "✕"}</td>
                <td className="px-4 py-2 text-right space-x-1 whitespace-nowrap">
                  <Btn variant="ghost" onClick={() => setEditing({ ...c })}>Edit</Btn>
                  <Btn variant="danger" onClick={() => remove(c)}>Delete</Btn>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {editing && (
        <Modal title={editing.id ? "Edit category" : "New category"} onClose={() => setEditing(null)}>
          <Field label="Name">
            <input className={inputClass} value={editing.name ?? ""} onChange={(e) => setEditing({ ...editing, name: e.target.value })} />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Parent">
              <select className={inputClass} value={editing.parent_id ?? ""} onChange={(e) => setEditing({ ...editing, parent_id: e.target.value ? Number(e.target.value) : null })}>
                <option value="">— none —</option>
                {items.filter((c) => c.id !== editing.id).map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </Field>
            <Field label="Position">
              <input type="number" className={inputClass} value={editing.position ?? 0} onChange={(e) => setEditing({ ...editing, position: Number(e.target.value) })} />
            </Field>
          </div>
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
