const API = "";
let token = localStorage.getItem("qs_token") || "";
let selectedId = null;
const $ = (id) => document.getElementById(id);
function setAuthMsg(t) { $("auth-msg").textContent = t || ""; }
function setDetailMsg(t) { $("detail-msg").textContent = t || ""; }
async function api(path, opts = {}) {
  const headers = { "Content-Type": "application/json", ...(opts.headers || {}) };
  if (token) headers["Authorization"] = "Bearer " + token;
  const res = await fetch(API + path, { ...opts, headers });
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = { raw: text }; }
  if (!res.ok) throw new Error((data && data.error) || res.statusText || "error");
  return data;
}
function showMain(yes) {
  $("auth-panel").classList.toggle("hidden", yes);
  $("main-panel").classList.toggle("hidden", !yes);
}
async function login() {
  try {
    const data = await api("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ email: $("email").value.trim(), password: $("password").value }),
    });
    token = data.token;
    localStorage.setItem("qs_token", token);
    setAuthMsg("登入成功");
    showMain(true);
    await refresh();
  } catch (e) { setAuthMsg(e.message); }
}
async function register() {
  try {
    await api("/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify({ email: $("email").value.trim(), password: $("password").value, display_name: "user" }),
    });
    setAuthMsg("註冊成功，請登入");
  } catch (e) { setAuthMsg(e.message); }
}
function logout() {
  token = ""; localStorage.removeItem("qs_token"); selectedId = null; showMain(false);
}
async function refresh() {
  const data = await api("/api/v1/instances");
  const box = $("instances");
  box.innerHTML = "";
  (data.items || []).forEach((it) => {
    const el = document.createElement("div");
    el.className = "item" + (selectedId === it.id ? " active" : "");
    el.innerHTML = `<span>#${it.id} · ${it.symbol} · ${it.template_id}</span><span class="badge ${it.status}">${it.status}</span>`;
    el.onclick = () => selectInstance(it);
    box.appendChild(el);
  });
}
function selectInstance(it) {
  selectedId = it.id;
  $("detail-empty").classList.add("hidden");
  $("detail").classList.remove("hidden");
  $("detail-json").textContent = JSON.stringify(it, null, 2);
  $("portfolio-json").textContent = "";
  setDetailMsg("");
  refresh();
}
async function createInst() {
  try {
    await api("/api/v1/instances", {
      method: "POST",
      body: JSON.stringify({ template_id: "lunar", symbol: $("symbol").value.trim(), capital_quota: Number($("capital").value) || 0 }),
    });
    await refresh();
  } catch (e) { setDetailMsg(e.message); }
}
async function startStop(action) {
  if (!selectedId) return;
  try {
    const it = await api(`/api/v1/instances/${selectedId}/${action}`, { method: "POST" });
    $("detail-json").textContent = JSON.stringify(it, null, 2);
    await refresh();
  } catch (e) { setDetailMsg(e.message); }
}
async function loadPortfolio() {
  if (!selectedId) return;
  try {
    const p = await api(`/api/v1/instances/${selectedId}/portfolio`);
    $("portfolio-json").textContent = JSON.stringify(p, null, 2);
  } catch (e) { setDetailMsg(e.message); }
}
async function sendTrade() {
  if (!selectedId) return;
  try {
    const r = await api(`/api/v1/instances/${selectedId}/trades`, {
      method: "POST",
      body: JSON.stringify({ side: $("side").value, engine: $("engine").value, qty: Number($("qty").value) }),
    });
    setDetailMsg("已下發: " + r.client_order_id);
  } catch (e) { setDetailMsg(e.message); }
}
$("btn-login").onclick = login;
$("btn-register").onclick = register;
$("btn-logout").onclick = logout;
$("btn-create").onclick = createInst;
$("btn-refresh").onclick = () => refresh().catch((e) => setDetailMsg(e.message));
$("btn-start").onclick = () => startStop("start");
$("btn-stop").onclick = () => startStop("stop");
$("btn-portfolio").onclick = loadPortfolio;
$("btn-trade").onclick = sendTrade;
if (token) { showMain(true); refresh().catch(() => logout()); }
