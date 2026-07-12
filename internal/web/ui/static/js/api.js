// Thin fetch wrapper around the ONTCM REST API. All methods return parsed JSON
// and throw an Error (with the server's message when available) on non-2xx.

const BASE = "/api/v1";

async function request(path, { method = "GET", body } = {}) {
  const opts = { method, headers: {} };
  if (body !== undefined) {
    opts.headers["Content-Type"] = "application/json";
    opts.body = JSON.stringify(body);
  }
  const res = await fetch(BASE + path, opts);
  const text = await res.text();
  let data = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = text;
    }
  }
  if (!res.ok) {
    const msg = (data && data.message) || `请求失败 (HTTP ${res.status})`;
    throw new Error(msg);
  }
  return data;
}

export const api = {
  // diagnostic workflow
  startSession: (patient = {}) => request("/diagnostic", { method: "POST", body: patient }),
  processStep: (id, step, answers) =>
    request(`/diagnostic/${id}/step`, { method: "POST", body: { step, answers } }),
  getSessionState: (id) => request(`/diagnostic/${id}/state`),
  endSession: (id) => request(`/diagnostic/${id}`, { method: "DELETE" }),

  // formula + herb lookups
  listFormulas: () => request("/formulas"),
  searchFormulas: (q) => request(`/formulas/search?q=${encodeURIComponent(q)}`),
  getFormula: (id) => request(`/formulas/${encodeURIComponent(id)}`),
  listHerbs: () => request("/herbs"),
  searchHerbs: (q) => request(`/herbs/search?q=${encodeURIComponent(q)}`),
  getHerb: (id) => request(`/herbs/${encodeURIComponent(id)}`),
};
