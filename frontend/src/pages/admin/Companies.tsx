import { useEffect, useState } from "react";
import { adminApi, type AdminCompany } from "../../api";
import { PageHeader, Modal, Field, inputClass, Btn } from "../../components/ui";

export function Companies() {
  const [items, setItems] = useState<AdminCompany[]>([]);
  const [search, setSearch] = useState("");
  const [editing, setEditing] = useState<AdminCompany | null>(null);
  const [error, setError] = useState("");

  function load() { adminApi.companies({ search }).then((r) => setItems(r.data)).catch(() => {}); }
  useEffect(() => { const t = setTimeout(load, 200); return () => clearTimeout(t); }, [search]);

  async function toggleVerify(c: AdminCompany) {
    await adminApi.updateCompany(c.id, { is_verified: !c.is_verified });
    load();
  }

  async function save() {
    if (!editing) return;
    setError("");
    try {
      await adminApi.updateCompany(editing.id, {
        name: editing.name,
        tax_number: editing.tax_number,
        country: editing.country,
        is_verified: editing.is_verified,
      });
      setEditing(null);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    }
  }

  return (
    <div>
      <PageHeader title="Companies (B2B)" />
      <input placeholder="Search…" value={search} onChange={(e) => setSearch(e.target.value)} className={inputClass + " max-w-xs mb-4"} />

      <div className="card overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-slate-500 text-left">
            <tr>
              <th className="px-4 py-2">Name</th>
              <th className="px-4 py-2">Tax #</th>
              <th className="px-4 py-2">Country</th>
              <th className="px-4 py-2 text-right">Users</th>
              <th className="px-4 py-2 text-right">Orders</th>
              <th className="px-4 py-2">Verified</th>
              <th className="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {items.map((c) => (
              <tr key={c.id} className="border-t border-slate-100">
                <td className="px-4 py-2 font-medium">{c.name}</td>
                <td className="px-4 py-2 text-slate-500">{c.tax_number ?? "—"}</td>
                <td className="px-4 py-2">{c.country ?? "—"}</td>
                <td className="px-4 py-2 text-right">{c.users_count ?? 0}</td>
                <td className="px-4 py-2 text-right">{c.orders_count ?? 0}</td>
                <td className="px-4 py-2">
                  <button onClick={() => toggleVerify(c)} className={`text-xs px-2 py-0.5 rounded ${c.is_verified ? "bg-green-100 text-green-700" : "bg-slate-100 text-slate-500"}`}>
                    {c.is_verified ? "verified" : "unverified"}
                  </button>
                </td>
                <td className="px-4 py-2 text-right">
                  <Btn variant="ghost" onClick={() => setEditing({ ...c })}>Edit</Btn>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {editing && (
        <Modal title="Edit company" onClose={() => setEditing(null)}>
          <Field label="Name">
            <input className={inputClass} value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Tax number">
              <input className={inputClass} value={editing.tax_number ?? ""} onChange={(e) => setEditing({ ...editing, tax_number: e.target.value })} />
            </Field>
            <Field label="Country (ISO-2)">
              <input maxLength={2} className={inputClass} value={editing.country ?? ""} onChange={(e) => setEditing({ ...editing, country: e.target.value.toUpperCase() })} />
            </Field>
          </div>
          <label className="flex items-center gap-2 text-sm mb-4">
            <input type="checkbox" checked={editing.is_verified} onChange={(e) => setEditing({ ...editing, is_verified: e.target.checked })} />
            Verified
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
