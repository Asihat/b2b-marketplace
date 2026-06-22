import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useStore } from "../store";
import { useT } from "../i18n";

export function Login() {
  const { login, register } = useStore();
  const t = useT();
  const navigate = useNavigate();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const [email, setEmail] = useState("buyer@acme.test");
  const [password, setPassword] = useState("password");
  const [name, setName] = useState("");
  const [type, setType] = useState<"b2c" | "b2b">("b2c");
  const [companyName, setCompanyName] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      if (mode === "login") {
        await login(email, password);
      } else {
        await register({ name, email, password, type, ...(type === "b2b" ? { company_name: companyName } : {}) });
      }
      navigate("/");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="max-w-md mx-auto mt-4">
      <div className="text-center mb-6">
        <h1 className="text-2xl font-bold text-slate-900">{mode === "login" ? t("login.welcomeBack") : t("login.createAccount")}</h1>
        <p className="text-sm text-slate-500 mt-1">
          {mode === "login" ? t("login.signinSub") : t("login.registerSub")}
        </p>
      </div>

      <div className="card p-6">
        <div className="grid grid-cols-2 gap-1 p-1 bg-slate-100 rounded-xl mb-5">
          {(["login", "register"] as const).map((m) => (
            <button
              key={m}
              onClick={() => setMode(m)}
              className={`py-2 rounded-lg text-sm font-medium transition ${
                mode === m ? "bg-white text-slate-900 shadow-sm" : "text-slate-500 hover:text-slate-700"
              }`}
            >
              {t(`login.${m}`)}
            </button>
          ))}
        </div>

        <form onSubmit={submit} className="space-y-3">
          {mode === "register" && (
            <>
              <input placeholder={t("login.fullName")} value={name} onChange={(e) => setName(e.target.value)} required className="input" />
              <div className="grid grid-cols-2 gap-1 p-1 bg-slate-100 rounded-xl">
                {(["b2c", "b2b"] as const).map((t) => (
                  <button type="button" key={t} onClick={() => setType(t)}
                    className={`py-2 rounded-lg text-sm uppercase font-medium transition ${
                      type === t ? "bg-white text-slate-900 shadow-sm" : "text-slate-500"
                    }`}>
                    {t}
                  </button>
                ))}
              </div>
              {type === "b2b" && (
                <input placeholder={t("login.companyName")} value={companyName} onChange={(e) => setCompanyName(e.target.value)} required className="input" />
              )}
            </>
          )}

          <input type="email" placeholder={t("login.email")} value={email} onChange={(e) => setEmail(e.target.value)} required className="input" />
          <input type="password" placeholder={t("login.password")} value={password} onChange={(e) => setPassword(e.target.value)} required className="input" />

          {error && <p className="text-sm text-rose-600 bg-rose-50 rounded-lg px-3 py-2">{error}</p>}

          <button disabled={busy} className="btn-primary w-full !py-2.5">
            {busy ? t("login.pleaseWait") : mode === "login" ? t("login.signInBtn") : t("login.createBtn")}
          </button>
        </form>
      </div>

      {mode === "login" && (
        <p className="text-xs text-slate-400 mt-4 text-center leading-relaxed">
          {t("login.demo")}<br />
          <code className="text-slate-500">buyer@acme.test</code> ·{" "}
          <code className="text-slate-500">customer@example.test</code> ·{" "}
          <code className="text-slate-500">admin@marketplace.test</code>
        </p>
      )}
    </div>
  );
}
