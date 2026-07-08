import { useEffect, useState } from "react";
import { adminApi, type AppSettings } from "../../api";
import { Btn, inputClass, PageHeader } from "../../components/ui";
import { useStore } from "../../store";

export function Settings() {
  const { refreshSettings } = useStore();
  const [settings, setSettings] = useState<AppSettings | null>(null);
  const [mode, setMode] = useState<AppSettings["mode"]>("b2c");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [icon, setIcon] = useState<File | null>(null);
  const [iconPreview, setIconPreview] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const [removing, setRemoving] = useState(false);
  const [companyName, setCompanyName] = useState("");
  const [companyDescription, setCompanyDescription] = useState("");
  const [savingBranding, setSavingBranding] = useState(false);

  useEffect(() => {
    adminApi.settings()
      .then((res) => {
        setSettings(res);
        setMode(res.mode);
        setCompanyName(res.company_name);
        setCompanyDescription(res.company_description);
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Settings could not be loaded."))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!icon) {
      setIconPreview(null);
      return;
    }
    const url = URL.createObjectURL(icon);
    setIconPreview(url);
    return () => URL.revokeObjectURL(url);
  }, [icon]);

  async function save() {
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      const next = await adminApi.saveSettings({ mode });
      setSettings(next);
      setMode(next.mode);
      await refreshSettings();
      setSuccess("Settings saved.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  async function uploadIcon() {
    if (!icon) return;
    setUploading(true);
    setError("");
    setSuccess("");
    try {
      const next = await adminApi.uploadIcon(icon);
      setSettings(next);
      await refreshSettings();
      setIcon(null);
      setSuccess("Main icon uploaded.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Icon upload failed");
    } finally {
      setUploading(false);
    }
  }

  async function removeIcon() {
    if (!settings?.icon_url || !window.confirm("Remove the current main icon?")) return;
    setRemoving(true);
    setError("");
    setSuccess("");
    try {
      const next = await adminApi.removeIcon();
      setSettings(next);
      setIcon(null);
      await refreshSettings();
      setSuccess("Main icon removed.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Icon removal failed");
    } finally {
      setRemoving(false);
    }
  }

  async function saveBranding() {
    setSavingBranding(true);
    setError("");
    setSuccess("");
    try {
      const next = await adminApi.saveSettings({
        company_name: companyName.trim(),
        company_description: companyDescription.trim(),
      });
      setSettings(next);
      setCompanyName(next.company_name);
      setCompanyDescription(next.company_description);
      await refreshSettings();
      setSuccess("Company details saved.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Company details could not be saved");
    } finally {
      setSavingBranding(false);
    }
  }

  return (
    <div>
      <PageHeader title="Settings" />

      {error && <p className="max-w-3xl text-sm text-rose-600 mb-3">{error}</p>}
      {success && <p className="max-w-3xl text-sm text-emerald-600 mb-3">{success}</p>}

      <div className="card p-5 max-w-3xl mb-6">
        <div className="flex flex-col gap-1 mb-5">
          <h2 className="text-base font-semibold text-slate-900">Company details</h2>
          <p className="text-sm text-slate-500">Shown in the app header, footer, browser title, and page description.</p>
        </div>
        {loading ? (
          <div className="py-6 text-sm text-slate-400">Loading settings...</div>
        ) : (
          <div className="space-y-4">
            <label className="block">
              <span className="block text-xs font-medium text-slate-500 mb-1.5">Company name</span>
              <input className={inputClass} maxLength={120} value={companyName} onChange={(e) => setCompanyName(e.target.value)} />
            </label>
            <label className="block">
              <span className="block text-xs font-medium text-slate-500 mb-1.5">Description</span>
              <textarea className={`${inputClass} min-h-24 resize-y`} maxLength={500} value={companyDescription} onChange={(e) => setCompanyDescription(e.target.value)} />
            </label>
            <div className="flex justify-end">
              <Btn
                onClick={saveBranding}
                disabled={savingBranding || !companyName.trim() || (companyName === settings?.company_name && companyDescription === settings?.company_description)}
              >
                {savingBranding ? "Saving..." : "Save company details"}
              </Btn>
            </div>
          </div>
        )}
      </div>

      <div className="card p-5 max-w-3xl mb-6">
        <div className="flex flex-col gap-1 mb-5">
          <h2 className="text-base font-semibold text-slate-900">Main icon</h2>
          <p className="text-sm text-slate-500">Used in the storefront, admin panel, and browser tab. PNG, JPG, or WebP up to 2 MB.</p>
        </div>

        {loading ? (
          <div className="py-6 text-sm text-slate-400">Loading settings...</div>
        ) : (
          <div className="flex flex-col sm:flex-row sm:items-center gap-4">
            <div className="w-20 h-20 rounded-2xl bg-slate-50 ring-1 ring-slate-200 grid place-items-center overflow-hidden shrink-0">
              {iconPreview || settings?.icon_url ? (
                <img src={iconPreview ?? settings?.icon_url ?? ""} alt="Main icon preview" className="w-full h-full object-contain" />
              ) : (
                <span className="text-2xl text-brand-600">◆</span>
              )}
            </div>
            <div className="flex-1">
              <input
                type="file"
                accept="image/png,image/jpeg,image/webp"
                onChange={(e) => setIcon(e.target.files?.[0] ?? null)}
                className="block w-full text-sm text-slate-500 file:mr-4 file:rounded-lg file:border-0 file:bg-slate-100 file:px-4 file:py-2 file:text-sm file:font-medium file:text-slate-700 hover:file:bg-slate-200"
              />
              <div className="mt-3 flex justify-end gap-2">
                {settings?.icon_url && (
                  <Btn onClick={removeIcon} variant="danger" disabled={uploading || removing}>
                    {removing ? "Removing..." : "Remove icon"}
                  </Btn>
                )}
                <Btn onClick={uploadIcon} disabled={!icon || uploading || removing}>
                  {uploading ? "Uploading..." : "Upload icon"}
                </Btn>
              </div>
            </div>
          </div>
        )}
      </div>

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
