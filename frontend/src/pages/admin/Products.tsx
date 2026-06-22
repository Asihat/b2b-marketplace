import { useEffect, useState } from "react";
import { adminApi, type AdminProduct, type AdminCategory } from "../../api";
import { PageHeader, Modal, Field, inputClass, Btn } from "../../components/ui";
import { useDebouncedValue } from "../../hooks/useDebouncedValue";

type Draft = Partial<AdminProduct>;

const EMPTY: Draft = {
  sku: "", name: "", brand: "", base_price: "0", stock: 0, min_order_qty: 1,
  is_b2b_only: false, is_active: true, category_id: null,
};

export function Products() {
  const [products, setProducts] = useState<AdminProduct[]>([]);
  const [categories, setCategories] = useState<AdminCategory[]>([]);
  const [search, setSearch] = useState("");
  const [editing, setEditing] = useState<Draft | null>(null);
  const [imagesText, setImagesText] = useState("");
  const [error, setError] = useState("");
  const debouncedSearch = useDebouncedValue(search);

  function load() {
    adminApi.products({ search: debouncedSearch }).then((r) => setProducts(r.data)).catch(() => {});
  }

  function openEditor(draft: Draft) {
    setImagesText((draft.images ?? []).map((i) => i.url).join("\n"));
    setError("");
    setEditing(draft);
  }

  useEffect(() => { adminApi.categories().then(setCategories).catch(() => {}); }, []);
  useEffect(() => { load(); }, [debouncedSearch]);

  async function save() {
    if (!editing) return;
    setError("");
    const images = imagesText.split("\n").map((s) => s.trim()).filter(Boolean);
    try {
      await adminApi.saveProduct(editing.id ?? null, {
        sku: editing.sku,
        name: editing.name,
        brand: editing.brand,
        base_price: Number(editing.base_price),
        stock: Number(editing.stock ?? 0),
        min_order_qty: Number(editing.min_order_qty ?? 1),
        category_id: editing.category_id || null,
        is_b2b_only: !!editing.is_b2b_only,
        is_active: !!editing.is_active,
        images,
      });
      setEditing(null);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    }
  }

  async function remove(p: AdminProduct) {
    if (!confirm(`Delete ${p.name}?`)) return;
    await adminApi.deleteProduct(p.id);
    load();
  }

  return (
    <div>
      <PageHeader
        title="Products"
        action={<Btn onClick={() => openEditor({ ...EMPTY })}>+ New product</Btn>}
      />

      <input
        placeholder="Search…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className={inputClass + " max-w-xs mb-4"}
      />

      <div className="card overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-slate-500 text-left">
            <tr>
              <th className="px-4 py-2 w-12"></th>
              <th className="px-4 py-2">SKU</th>
              <th className="px-4 py-2">Name</th>
              <th className="px-4 py-2 text-right">Price</th>
              <th className="px-4 py-2 text-right">Stock</th>
              <th className="px-4 py-2">Flags</th>
              <th className="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {products.map((p) => (
              <tr key={p.id} className="border-t border-slate-100">
                <td className="px-2 py-2">
                  <div className="w-9 h-9 rounded bg-slate-100 overflow-hidden">
                    {p.images?.[0] && <img src={p.images[0].url} alt="" className="w-full h-full object-cover" />}
                  </div>
                </td>
                <td className="px-4 py-2 font-mono text-xs">{p.sku}</td>
                <td className="px-4 py-2">{p.name}<div className="text-xs text-slate-400">{p.brand}</div></td>
                <td className="px-4 py-2 text-right">{Number(p.base_price).toFixed(2)}</td>
                <td className="px-4 py-2 text-right">{p.stock}</td>
                <td className="px-4 py-2 text-xs space-x-1">
                  {p.is_b2b_only && <span className="bg-amber-100 text-amber-700 px-1.5 py-0.5 rounded">B2B</span>}
                  {!p.is_active && <span className="bg-slate-200 text-slate-500 px-1.5 py-0.5 rounded">hidden</span>}
                </td>
                <td className="px-4 py-2 text-right space-x-1 whitespace-nowrap">
                  <Btn variant="ghost" onClick={() => openEditor({ ...p })}>Edit</Btn>
                  <Btn variant="danger" onClick={() => remove(p)}>Delete</Btn>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {editing && (
        <Modal title={editing.id ? "Edit product" : "New product"} onClose={() => setEditing(null)}>
          <div className="grid grid-cols-2 gap-3">
            <Field label="SKU">
              <input className={inputClass} value={editing.sku ?? ""} onChange={(e) => setEditing({ ...editing, sku: e.target.value })} />
            </Field>
            <Field label="Brand">
              <input className={inputClass} value={editing.brand ?? ""} onChange={(e) => setEditing({ ...editing, brand: e.target.value })} />
            </Field>
          </div>
          <Field label="Name">
            <input className={inputClass} value={editing.name ?? ""} onChange={(e) => setEditing({ ...editing, name: e.target.value })} />
          </Field>
          <div className="grid grid-cols-3 gap-3">
            <Field label="Base price">
              <input type="number" step="0.01" className={inputClass} value={editing.base_price ?? ""} onChange={(e) => setEditing({ ...editing, base_price: e.target.value })} />
            </Field>
            <Field label="Stock">
              <input type="number" className={inputClass} value={editing.stock ?? 0} onChange={(e) => setEditing({ ...editing, stock: Number(e.target.value) })} />
            </Field>
            <Field label="Min order qty">
              <input type="number" className={inputClass} value={editing.min_order_qty ?? 1} onChange={(e) => setEditing({ ...editing, min_order_qty: Number(e.target.value) })} />
            </Field>
          </div>
          <Field label="Category">
            <select className={inputClass} value={editing.category_id ?? ""} onChange={(e) => setEditing({ ...editing, category_id: e.target.value ? Number(e.target.value) : null })}>
              <option value="">— none —</option>
              {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </select>
          </Field>
          <Field label="Image URLs (one per line — first is primary)">
            <textarea
              rows={3}
              className={inputClass + " font-mono text-xs"}
              placeholder="https://…/image1.jpg&#10;https://…/image2.jpg"
              value={imagesText}
              onChange={(e) => setImagesText(e.target.value)}
            />
          </Field>
          {imagesText.trim() && (
            <div className="flex gap-2 mb-3">
              {imagesText.split("\n").map((s) => s.trim()).filter(Boolean).slice(0, 5).map((url, i) => (
                <div key={i} className="w-12 h-12 rounded bg-slate-100 overflow-hidden">
                  <img src={url} alt="" className="w-full h-full object-cover" />
                </div>
              ))}
            </div>
          )}
          <div className="flex gap-4 mb-4 text-sm">
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={!!editing.is_b2b_only} onChange={(e) => setEditing({ ...editing, is_b2b_only: e.target.checked })} />
              B2B only
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
