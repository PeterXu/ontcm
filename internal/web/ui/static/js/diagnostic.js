// Diagnostic wizard. Drives the 12-step session from the browser:
//   start → render each step's self-describing `question` payload →
//   auto-advance the reasoning steps (6–11) → render the final prescription.
//
// All progression logic comes from the API response (current_step + question
// shape); this module only renders what the server returns.

import { api } from "./api.js";

// Presentational labels for the 12-step stepper. The 12 steps are a fixed
// domain concept; the *logic* (which step to POST, what to render) is driven by
// the API response's current_step + question payload, not by this list.
const STEP_LABELS = [
  "主诉", "急症", "十问", "舌诊", "脉诊",
  "定经", "方证", "药证", "证据", "反验", "合病", "选方",
];

let session = null;

export function initDiagnostic() {
  const stage = document.getElementById("diag-stage");
  const stepper = document.getElementById("stepper");
  renderStart(stage, stepper);
}

// ---- entry / restart ----

function renderStart(stage, stepper) {
  stepper.hidden = true;
  session = null;
  stage.innerHTML = `
    <div class="card hero">
      <h2>开始一次六经辨证</h2>
      <p>依次录入主诉、十问、舌象与脉象，系统将自动完成定经、方证对勘、药证校验等推理，并给出选方建议。</p>
      <button class="btn btn-lg" id="start-btn">开始辨证</button>
    </div>`;
  document.getElementById("start-btn").addEventListener("click", () => start(stage, stepper));
}

async function start(stage, stepper) {
  try {
    setBusy(stage, "正在创建会话…");
    session = await api.startSession({});
    renderStepper(stepper, session.current_step);
    renderStep(stage, stepper, session);
  } catch (e) {
    showError(stage, e.message, () => renderStart(stage, stepper));
  }
}

// ---- step router ----

function renderStep(stage, stepper, s) {
  if (s.emergency_halt) return renderHalt(stage, s);
  if (s.diagnostic_result) return renderResult(stage, s);
  // Steps 6–11 carry no question (pure reasoning) → auto-advance.
  if (!s.question) return runReasoning(stage, stepper, s);
  if (Array.isArray(s.question)) return renderCategoryForm(stage, stepper, s); // step 3
  return renderTemplateForm(stage, stepper, s); // steps 1, 4, 5
}

// ---- reasoning auto-advance (steps 6 → 12) ----

async function runReasoning(stage, stepper, s) {
  try {
    while (
      s.status === "active" &&
      !s.diagnostic_result &&
      !s.emergency_halt &&
      s.current_step <= 12
    ) {
      setBusy(stage, `正在分析 · ${labelFor(s.current_step)}（第 ${s.current_step} 步）`);
      s = await api.processStep(s.session_id, s.current_step, {});
      session = s;
      renderStepper(stepper, s.current_step);
    }
    renderStep(stage, stepper, s);
  } catch (e) {
    showError(stage, e.message, () => renderStep(stage, stepper, s));
  }
}

// ---- forms ----

function renderTemplateForm(stage, stepper, s) {
  const q = s.question;
  stage.innerHTML = `
    <div class="card">
      <h3 class="form-title">${q.title}</h3>
      ${q.description ? `<p class="form-desc">${q.description}</p>` : ""}
      ${q.instructions ? `<div class="form-instr">${q.instructions}</div>` : ""}
      <div id="form-error"></div>
      <form id="step-form" novalidate>
        ${q.fields.map(renderField).join("")}
        <div class="form-actions"><button type="submit" class="btn">下一步</button></div>
      </form>
    </div>`;
  wireForm(stage, stepper, s);
}

function renderCategoryForm(stage, stepper, s) {
  const cats = s.question;
  const body = cats.map((cat) => `
    <div class="category">
      <h4><span class="cat-icon">${cat.icon || ""}</span>${cat.name}</h4>
      <div class="cat-grid">${cat.questions.map(renderField).join("")}</div>
    </div>`).join("");
  stage.innerHTML = `
    <div class="card">
      <h3 class="form-title">十问为纲</h3>
      <p class="form-desc">逐项选择患者情况，正常项可留空。</p>
      <div id="form-error"></div>
      <form id="step-form" novalidate>
        ${body}
        <div class="form-actions"><button type="submit" class="btn">下一步</button></div>
      </form>
    </div>`;
  wireForm(stage, stepper, s);
}

function wireForm(stage, stepper, s) {
  document.getElementById("step-form").addEventListener("submit", (e) => {
    e.preventDefault();
    submit(stage, stepper, s);
  });
}

// Render a single field by its self-describing type. No field-ID special-casing.
function renderField(f) {
  const req = f.required ? '<span class="req">*</span>' : "";
  const help = f.help_text ? `<div class="help">${f.help_text}</div>` : "";
  const ph = f.placeholder || "";
  let control;
  switch (f.type) {
    case "textarea":
      control = `<textarea data-id="${f.id}" data-type="${f.type}" placeholder="${ph}"></textarea>`;
      break;
    case "select":
      control = `<select data-id="${f.id}" data-type="${f.type}">
        <option value="">（请选择）</option>
        ${f.options.map((o) => `<option value="${o}">${o}</option>`).join("")}
      </select>`;
      break;
    case "multiselect":
      control = `<div class="checks" data-id="${f.id}" data-type="${f.type}">
        ${f.options.map((o) => `<label class="check"><input type="checkbox" value="${o}"> ${o}</label>`).join("")}
      </div>`;
      break;
    case "number":
      control = `<input type="number" data-id="${f.id}" data-type="${f.type}" placeholder="${ph}">`;
      break;
    default:
      control = `<input type="text" data-id="${f.id}" data-type="${f.type}" placeholder="${ph}">`;
  }
  return `<div class="field"><label>${f.label}${req}</label>${control}${help}</div>`;
}

function collectAnswers() {
  const answers = {};
  document.querySelectorAll("#step-form [data-id]").forEach((el) => {
    const { id, type } = el.dataset;
    if (type === "multiselect") {
      const checked = Array.from(el.querySelectorAll("input:checked")).map((c) => c.value);
      if (checked.length) answers[id] = checked;
    } else if (type === "number") {
      if (el.value !== "") answers[id] = Number(el.value);
    } else if (el.value !== "") {
      answers[id] = el.value;
    }
  });
  return answers;
}

function missingRequired() {
  const missing = [];
  document.querySelectorAll("#step-form .field").forEach((field) => {
    const label = field.querySelector("label");
    if (!label || !label.querySelector(".req")) return;
    const control = field.querySelector("[data-id]");
    if (!control) return;
    let empty;
    if (control.dataset.type === "multiselect") {
      empty = !control.querySelector("input:checked");
    } else {
      empty = control.value === "";
    }
    if (empty) missing.push(label.textContent.replace("*", "").trim());
  });
  return missing;
}

async function submit(stage, stepper, s) {
  clearFormError();
  const missing = missingRequired();
  if (missing.length) {
    formError(`请填写必填项：${missing.join("、")}`);
    return;
  }
  const answers = collectAnswers();
  const btn = document.querySelector('#step-form button[type="submit"]');
  btn.disabled = true;
  btn.textContent = "提交中…";
  try {
    const next = await api.processStep(s.session_id, s.current_step, answers);
    session = next;
    renderStepper(stepper, next.current_step);
    renderStep(stage, stepper, next);
  } catch (e) {
    btn.disabled = false;
    btn.textContent = "下一步";
    formError(e.message);
  }
}

// ---- result / halt ----

function renderResult(stage, s) {
  const r = s.diagnostic_result;
  const matched = (r.matched_symptoms || []).map((x) => `<span class="tag match">${esc(x)}</span>`).join("");
  const contra = (r.contraindications || []).map((x) => `<span class="tag">${esc(x)}</span>`).join("");
  stage.innerHTML = `
    <div class="card">
      <span class="result-meridian">${esc(r.meridian)}</span>
      <h3 class="result-formula">${esc(r.selected_formula)}</h3>
      <div class="kv"><span class="k">证据强度</span><span>${r.evidence_score} · ${r.is_reliable ? "诊断可靠" : "证据不足"}</span></div>
      <div class="kv"><span class="k">方剂编号</span><span>${esc(r.formula_id)}</span></div>
      ${matched ? `<div class="detail-block"><h4>对应症状</h4><div class="tags">${matched}</div></div>` : ""}
      ${contra ? `<div class="detail-block"><h4>用药提醒</h4><div class="tags">${contra}</div></div>` : ""}
      ${r.llm_refinement_reason ? `<div class="llm-reason"><strong>模型细选：</strong>${esc(r.llm_refinement_reason)}</div>` : ""}
      <div class="form-actions" style="margin-top:20px"><button class="btn btn-ghost" id="restart-btn">重新辨证</button></div>
    </div>`;
  document.getElementById("restart-btn").addEventListener("click", () => renderStart(stage, document.getElementById("stepper")));
}

function renderHalt(stage, s) {
  stage.innerHTML = `
    <div class="card">
      <div class="banner error"><strong>疑似急危重症，已中止辨证</strong></div>
      <p>${esc(s.emergency_reason || "")}</p>
      <p class="muted">请立即转诊或前往急诊处理，切勿延误。</p>
      <div class="form-actions"><button class="btn btn-ghost" id="restart-btn">重新开始</button></div>
    </div>`;
  document.getElementById("restart-btn").addEventListener("click", () => renderStart(stage, document.getElementById("stepper")));
}

// ---- helpers ----

function renderStepper(stepper, currentStep) {
  stepper.hidden = false;
  stepper.innerHTML = STEP_LABELS.map((label, i) => {
    const n = i + 1;
    let cls = "step-node";
    if (n < currentStep) cls += " done";
    else if (n === currentStep) cls += " current";
    return `<div class="${cls}"><div class="step-dot">${n}</div><span class="step-label">${label}</span></div>`;
  }).join("");
}

function labelFor(step) {
  return STEP_LABELS[step - 1] || `第 ${step} 步`;
}

function setBusy(stage, msg) {
  stage.innerHTML = `<div class="thinking"><div class="spinner"></div><span>${esc(msg)}</span></div>`;
}

function showError(stage, msg, retry) {
  stage.innerHTML = `
    <div class="card">
      <div class="banner error">${esc(msg)}</div>
      <div class="form-actions"><button class="btn btn-ghost" id="retry-btn">重试</button></div>
    </div>`;
  document.getElementById("retry-btn").addEventListener("click", retry);
}

function formError(msg) {
  const el = document.getElementById("form-error");
  if (el) el.innerHTML = `<div class="banner error">${esc(msg)}</div>`;
}
function clearFormError() {
  const el = document.getElementById("form-error");
  if (el) el.innerHTML = "";
}

function esc(s) {
  return String(s).replace(/[&<>"']/g, (c) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  }[c]));
}
