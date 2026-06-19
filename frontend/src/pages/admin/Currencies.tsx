import { useEffect, useState } from "react";
import { adminApi, type Currency } from "../../api";
import { PageHeader, Modal, Field, inputClass, Btn } from "../../components/ui";

type Row = Currency & { id: number; is_active: boolean };
type Draft = Partial<Row>;
const EMPTY: Draft = { code: "", name: "", symbol: "", exchange_rate: "1", is_base: false, is_active: true };

export function Currencies() {
  const [items, setItems] = useState<Row[]>([]);
  const [editing, setEditing] = useState<Draft | null>(null);
  const [error, setError] = useState("");

  function load() { adminApi.currencies().then(setItems).catch(() => {}); }
  useEffect(() => { load(); }, []);

  async function save() {
    if (!editing) return;
    setError("");
    try {
      await adminApi.saveCurrency(editing.id ?? null, {
        code: editing.code,
        name: editing.name,
        symbol: editing.symbol,
        exchange_rate: Number(editing.exchange_rate),
        is_base: !!editing.is_base,
        is_active: !!editing.is_active,
      });
      setEditing(null);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    }
  }

  async function remove(c: Row) {
    if (!confirm(`Delete ${c.code}?`)) return;
    try {
      await adminApi.deleteCurrency(c.id);
      load();
    } catch (e) {
      alert(e instanceof Error ? e.message : "Delete failed");
    }
  }

  return (
    <div>
      <PageHeader title="Currencies" action={<Btn onClick={() => setEditing({ ...EMPTY })}>+ New currency</Btn>} />
      <p className="text-xs text-slate-400 mb-4">Exchange rate is relative to the base currency (base = 1.0).</p>

      <div className="card overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-slate-500 text-left">
            <tr>
              <th className="px-4 py-2">Code</th>
              <th className="px-4 py-2">Name</th>
              <th className="px-4 py-2">Symbol</th>
              <th className="px-4 py-2 text-right">Rate</th>
              <th className="px-4 py-2">Flags</th>
              <th className="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {items.map((c) => (
              <tr key={c.id} className="border-t border-slate-100">
                <td className="px-4 py-2 font-mono font-medium">{c.code}</td>
                <td className="px-4 py-2">{c.name}</td>
                <td className="px-4 py-2">{c.symbol}</td>
                <td className="px-4 py-2 text-right">{Number(c.exchange_rate).toString()}</td>
                <td className="px-4 py-2 text-xs space-x-1">
                  {c.is_base && <span className="bg-indigo-100 text-indigo-700 px-1.5 py-0.5 rounded">base</span>}
                  {!c.is_active && <span className="bg-slate-200 text-slate-500 px-1.5 py-0.5 rounded">off</span>}
                </td>
                <td className="px-4 py-2 text-right space-x-1 whitespace-nowrap">
                  <Btn variant="ghost" onClick={() => setEditing({ ...c })}>Edit</Btn>
                  {!c.is_base && <Btn variant="danger" onClick={() => remove(c)}>Delete</Btn>}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {editing && (
        <Modal title={editing.id ? "Edit currency" : "New currency"} onClose={() => setEditing(null)}>
          <div className="grid grid-cols-3 gap-3">
            <Field label="Code">
              <input maxLength={3} className={inputClass} value={editing.code ?? ""} onChange={(e) => setEditing({ ...editing, code: e.target.value.toUpperCase() })} />
            </Field>
            <Field label="Symbol">
              <input className={inputClass} value={editing.symbol ?? ""} onChange={(e) => setEditing({ ...editing, symbol: e.target.value })} />
            </Field>
            <Field label="Rate">
              <input type="number" step="0.00000001" disabled={!!editing.is_base} className={inputClass} value={editing.exchange_rate ?? ""} onChange={(e) => setEditing({ ...editing, exchange_rate: e.target.value })} />
            </Field>
          </div>
          <Field label="Name">
            <input className={inputClass} value={editing.name ?? ""} onChange={(e) => setEditing({ ...editing, name: e.target.value })} />
          </Field>
          <div className="flex gap-4 mb-4 text-sm">
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={!!editing.is_base} onChange={(e) => setEditing({ ...editing, is_base: e.target.checked })} />
              Base currency
            </label>
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={!!editing.is_active} onChange={(e) => setEditing({ ...editing, is_active: e.target.checked })} />
              Active
            </label>
          </div>
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
