// App entry: tab router + lazy-loaded views. The diagnostic wizard mounts
// immediately; the two lookup views fetch their lists on first activation.

import { initDiagnostic } from "./diagnostic.js";
import { initLookup } from "./lookup.js";

const tabs = document.querySelectorAll(".tab");
const views = document.querySelectorAll(".view");

const loaded = { formulas: false, herbs: false };

function activate(name) {
  tabs.forEach((t) => t.classList.toggle("active", t.dataset.view === name));
  views.forEach((v) => v.classList.toggle("active", v.id === "view-" + name));

  if (name === "formulas" && !loaded.formulas) {
    initLookup(document.querySelector('#view-formulas .lookup')).loadList();
    loaded.formulas = true;
  } else if (name === "herbs" && !loaded.herbs) {
    initLookup(document.querySelector('#view-herbs .lookup')).loadList();
    loaded.herbs = true;
  }
}

tabs.forEach((t) => t.addEventListener("click", () => activate(t.dataset.view)));

initDiagnostic();
