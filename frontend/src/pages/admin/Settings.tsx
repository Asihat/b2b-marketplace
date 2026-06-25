import { useEffect, useState } from "react";
import { adminApi, type AppSettings } from "../../api";
import { Btn, PageHeader } from "../../components/ui";
import { useStore } from "../../store";

export function Settings() {
  const { refreshSettings } = useStore();
  const [settings, setSettings] = useState<AppSettings | null>(null);
  const [mode, setMode] = useState<AppSettings["mode"]>("b2c");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    adminApi.settings()
      .then((res) => {
        setSettings(res);
        setMode(res.mode);
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Settings could not be loaded."))
      .finally(() => setLoading(false));
  }, []);

  async function save() {
    setSaving(true);
    setError("");
    setSaved(false);
    try {
      const next = await adminApi.saveSettings({ mode });
      setSettings(next);
      setMode(next.mode);
      await refreshSettings();
      setSaved(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div>
      <PageHeader title="Settings" />

      <div className="card p-5 max-w-3xl">
        <div className="flex flex-col gap-1 mb-5">
          <h2 className="text-base font-semibold text-slate-900">Website mode</h2>
          <p className="text-sm text-slate-500">
            Controls storefront behavior and whether seller company names appear on product pages.
          </p>
        </div>

        {loading ? (
          <div className="py-10 text-sm text-slate-400">Loading settings...</div>
        ) : (
          <>
            <div className="grid sm:grid-cols-2 gap-3 mb-5">
              <button
                type="button"
                onClick={() => setMode("b2c")}
                className={`text-left rounded-lg p-4 ring-1 transition ${
                  mode === "b2c" ? "bg-brand-50 ring-brand-300" : "bg-white ring-slate-200 hover:bg-slate-50"
                }`}
              >
                <span className="block text-sm font-semibold text-slate-900">B2C</span>
                <span className="block text-sm text-slate-500 mt-1">Retail storefront. Seller company names are hidden.</span>
              </button>
              <button
                type="button"
                onClick={() => setMode("b2b")}
                className={`text-left rounded-lg p-4 ring-1 transition ${
                  mode === "b2b" ? "bg-brand-50 ring-brand-300" : "bg-white ring-slate-200 hover:bg-slate-50"
                }`}
              >
                <span className="block text-sm font-semibold text-slate-900">B2B</span>
                <span className="block text-sm text-slate-500 mt-1">Business marketplace. Seller company names are shown.</span>
              </button>
            </div>

            <div className="rounded-lg bg-slate-50 ring-1 ring-slate-200 p-3 text-sm text-slate-600 mb-5">
              Current saved mode: <span className="font-semibold uppercase text-slate-900">{settings?.mode}</span>
            </div>

            {error && <p className="text-sm text-rose-600 mb-3">{error}</p>}
            {saved && <p className="text-sm text-emerald-600 mb-3">Settings saved.</p>}

            <div className="flex justify-end">
              <Btn onClick={save} disabled={saving || mode === settings?.mode}>
                {saving ? "Saving..." : "Save settings"}
              </Btn>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
