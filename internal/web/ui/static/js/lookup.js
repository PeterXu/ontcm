// Formula + herb lookup. One controller drives both — they share a
// list/filter/detail (master-detail) pattern; only the field shapes differ.
//
// Search is client-side over the full cached list (name + keywords). The lists
// are small (112 formulas / 54 herbs), so this is instant and — unlike the
// /formulas/search endpoint, which indexes symptoms only — actually finds a
// formula by its name (e.g. "理中" → 理中汤).
//
// NOTE on field casing: formula nested types (HerbDose/FormulaSymptom/DrugSyndrome)
// ship with lowercase JSON tags, but herb nested types (HerbProperties/
// HerbDrugSyndrome/SafetyInfo) have no JSON tags and therefore marshal with
// Capitalized Go field names. The two detail renderers below reflect that.

import { api } from "./api.js";

export function initLookup(rootEl) {
  const kind = rootEl.dataset.kind; // "formula" | "herb"
  const $ = (role) => rootEl.querySelector(`[data-role="${role}"]`);
  const searchInput = $("search");
  const listEl = $("list");
  const detailEl = $("detail");
  const hintEl = $("hint");

  const endpoints = kind === "formula"
    ? { list: api.listFormulas, get: api.getFormula }
    : { list: api.listHerbs, get: api.getHerb };

  let cache = []; // [{id, name, sub, filterText}]
  let selectedId = null;

  async function load() {
    selectedId = null;
    searchInput.value = "";
    hintEl.textContent = "加载中…";
    listEl.innerHTML = "";
    detailEl.innerHTML = `<p class="muted">在左侧选择以查看详情。</p>`;
    try {
      const data = await endpoints.list();
      cache = toItems(data);
      renderList(cache);
      hintEl.textContent = `共 ${cache.length} 条`;
    } catch (e) {
      hintEl.textContent = "";
      detailEl.innerHTML = `<div class="banner error">${esc(e.message)}</div>`;
    }
  }

  // Instant client-side filter over the cached list.
  function doFilter() {
    const q = searchInput.value.trim().toLowerCase();
    if (!q) {
      renderList(cache);
      hintEl.textContent = `共 ${cache.length} 条`;
      return;
    }
    const hits = cache.filter((it) => it.filterText.includes(q));
    renderList(hits);
    hintEl.textContent = `${hits.length} / ${cache.length} 条 · "${searchInput.value.trim()}"`;
  }

  function toItems(data) {
    if (kind === "formula") {
      return data.formulas.map((f) => ({
        id: f.id,
        name: f.name,
        sub: `${f.meridian}${(f.key_symptoms || []).length ? " · " + f.key_symptoms.slice(0, 2).join("、") : ""}`,
        filterText: [f.id, f.name, f.meridian, ...(f.key_symptoms || [])].join(" ").toLowerCase(),
      }));
    }
    return data.herbs.map((h) => ({
      id: h.id,
      name: h.name,
      sub: `${h.tier}${h.nature ? " · " + h.nature : ""}`,
      filterText: [h.id, h.name, h.tier, h.nature, ...(h.main_meridians || [])].join(" ").toLowerCase(),
    }));
  }

  function renderList(items) {
    if (!items.length) {
      listEl.innerHTML = `<p class="muted">无结果。</p>`;
      return;
    }
    listEl.innerHTML = items.map((it) => `
      <div class="list-item ${selectedId === it.id ? "active" : ""}" data-id="${esc(it.id)}">
        <div class="li-name">${esc(it.name)}</div>
        <div class="li-sub">${esc(it.sub)}</div>
      </div>`).join("");
    listEl.querySelectorAll(".list-item").forEach((node) => {
      node.addEventListener("click", () => select(node.dataset.id));
    });
  }

  async function select(id) {
    selectedId = id;
    listEl.querySelectorAll(".list-item").forEach((n) => {
      n.classList.toggle("active", n.dataset.id === id);
    });
    detailEl.innerHTML = `<div class="thinking"><div class="spinner"></div><span>加载详情…</span></div>`;
    try {
      const item = await endpoints.get(id);
      detailEl.innerHTML = kind === "formula" ? formulaDetail(item) : herbDetail(item);
    } catch (e) {
      detailEl.innerHTML = `<div class="banner error">${esc(e.message)}</div>`;
    }
  }

  searchInput.addEventListener("input", doFilter);
  $("search-btn").addEventListener("click", doFilter);
  // "全部…" clears the filter and shows the whole cached list (no refetch).
  $("list-btn").addEventListener("click", () => {
    searchInput.value = "";
    doFilter();
  });

  return { loadList: load };
}

// ---- formula detail (lowercase JSON tags) ----

function formulaDetail(f) {
  const comp = (f.composition || []).map((c) => `
    <div class="composition-row">
      <span>${esc(c.name)}${c.processing ? ` <small class="muted">${esc(c.processing)}</small>` : ""}</span>
      <span>${esc(c.dose_original)}${c.dose_grams ? `（≈${c.dose_grams}g）` : ""}</span>
    </div>`).join("");

  const keySyms = (f.key_symptoms || []).map((s) => `
    <div class="kv"><span class="k">${s.required ? "必" : "或"}</span>
      <span><strong>${esc(s.name)}</strong>${s.clinical_sign ? " — " + esc(s.clinical_sign) : ""}</span></div>`).join("");

  const drugSyn = (f.drug_syndromes || []).map((d) => `
    <div class="kv"><span class="k">${esc(d.herb_name)}</span>
      <span>${esc(d.effect)} → ${esc(d.target_symptom)}</span></div>`).join("");

  const contra = (f.contraindications || []).map((c) => `<span class="tag">${esc(c)}</span>`).join("");

  return `
    <h3>${esc(f.name)} <small class="muted">${esc(f.meridian)}</small></h3>
    ${f.original_text ? `<div class="form-instr">原文：${esc(f.original_text)}</div>` : ""}
    ${comp ? section("组成", comp) : ""}
    ${keySyms ? section("主症", keySyms) : ""}
    ${drugSyn ? section("药证", drugSyn) : ""}
    ${f.preparation ? section("煮服法", `<p>${esc(f.preparation)}</p>`) : ""}
    ${contra ? section("禁忌", `<div class="tags">${contra}</div>`) : ""}
  `;
}

// ---- herb detail (Capitalized nested fields — no JSON tags on the model) ----

function herbDetail(h) {
  const p = h.properties || {};
  const taste = (p.Taste || []).join("、");
  const effect = (p.Effect || []).join("；");

  const ds = (h.drug_syndromes || []).map((d) => `
    <div class="kv"><span class="k">${esc(d.Effect)}</span>
      <span>${esc(d.Symptom)}${d.ExampleFormula ? `（${esc(d.ExampleFormula)}）` : ""}</span></div>`).join("");

  const s = h.safety || {};
  const safetyBits = [];
  if (s.ToxicityLevel) safetyBits.push(`毒性：${s.ToxicityLevel}`);
  if (s.MaxDose) safetyBits.push(`最大量：${s.MaxDose}g`);
  if (s.PregnancyWarning) safetyBits.push(`孕妇：${s.PregnancyWarning}`);
  if (s.ChildrenWarning) safetyBits.push(`儿童：${s.ChildrenWarning}`);
  if (s.ElderlyWarning) safetyBits.push(`老人：${s.ElderlyWarning}`);

  const pairTags = (h.common_pairings || []).map((c) => `<span class="tag">${esc(c)}</span>`).join("");
  const contra = (h.contraindications || []).map((c) => `<span class="tag">${esc(c)}</span>`).join("");

  return `
    <h3>${esc(h.name)} <small class="muted">${esc(h.tier)}</small></h3>
    <div class="kv"><span class="k">性味</span><span>${esc(p.Nature || "")}${taste ? " · " + taste : ""}${p.Direction ? " · " + esc(p.Direction) : ""}</span></div>
    ${effect ? `<div class="kv"><span class="k">功效</span><span>${esc(effect)}</span></div>` : ""}
    ${(h.main_meridians || []).length ? `<div class="kv"><span class="k">归经</span><span>${h.main_meridians.map(esc).join("、")}</span></div>` : ""}
    ${ds ? section("药证", ds) : ""}
    ${pairTags ? section("常用配伍", `<div class="tags">${pairTags}</div>`) : ""}
    ${safetyBits.length ? section("安全信息", `<div class="tags">${safetyBits.map((b) => `<span class="tag">${esc(b)}</span>`).join("")}</div>`) : ""}
    ${contra ? section("禁忌", `<div class="tags">${contra}</div>`) : ""}
  `;
}

// ---- helpers ----

function section(title, inner) {
  return `<div class="detail-block"><h4>${title}</h4>${inner}</div>`;
}

function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, (c) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  }[c]));
}
