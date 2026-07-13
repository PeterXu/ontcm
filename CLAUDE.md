# ONTCM - Traditional Chinese Medicine Diagnostic Agent

## Project Overview
Go-based TCM diagnostic agent implementing 六经辨证 (Six Meridians diagnosis) with REST API.

## Architecture
- **Knowledge Layer**: `internal/knowledge/` - Loader, InvertedIndex for formulas/herbs
- **Agent Layer**: `internal/agent/` - 12-step diagnostic state machine
- **Web Layer**: `internal/web/` - REST API handlers, session management, middleware
- **Models**: `internal/knowledge/models/` - Domain entities (Formula, Herb, DiagnosticSession)

## Chinese Language Support
- UTF-8 encoding throughout
- Chinese characters in domain models, API responses, documentation
- Field names match Chinese domain (e.g., `CoatingColor` not `Coating`)
- Question templates extracted from `docs/diagnosis_guide.md`

## Key Patterns

### Diagnostic Workflow
- 12-step state machine (steps 1-12)
- Step 2 is emergency check (skipped in progression: 1→3)
- Each step has executor function in `step_executor.go`
- Question templates in `question_templates.go`

### Session Management
- In-memory store with mutex locks (`internal/web/session/store.go`)
- 30-minute timeout
- Auto-cleanup goroutine

### API Pattern
```
Handler → Agent → StepExecutor → Knowledge Layer
```

### Data Flow
```
docs/*.md → Loader → InvertedIndex → Agent → Handler → REST API
```

## Important Commands
- `go build ./cmd/server` - Build server
- `go test ./... -v` - Run all tests
- `curl -s http://localhost:8080/api/v1/... | jq '.'` - Test API
- `pkill -f ontcm-server` - Kill running server

## Knowledge Base
- **Location**: `./docs`
- **Formulas**: 107 loaded from `docs/formulas/shanghanlun/` (108 unique source files; the 承气汤类 aggregate overview is excluded as a non-formula — its members 大/小/调胃承气汤 each have their own doc. 桂枝加大黄汤 + three filename-spelling duplicates consolidated. `index.md` files are skipped as navigation, not formulas.)
- **Herbs**: 54 total in `docs/herbs/`
- **Load at startup**: `loader.LoadAll()` + `index.BuildIndex(loader)`

## Diagnostic Process Steps
1. 主诉与病史 - Patient info
2. 急危重症排除 - Emergency check (validation gate)
3. 十问为纲 - Systematic inquiry (10 categories)
4. 舌诊 - Tongue diagnosis
5. 脉诊 - Pulse diagnosis
6. 定经 - Meridian determination
7. 方证对勘 - Formula matching
8. 药证校验 - Herb verification
9. 证据核查 - Evidence counting
10. 反向验证 - Contradiction check
11. 合病排查 - Combined disease
12. 选方定药 - Formula selection

## Security Features
- Input validation middleware (max 100 chars path, 500 chars query)
- XSS filtering in `middleware/validation.go`
- Security headers in `middleware/security.go`
- Structured logging with request IDs
- Graceful shutdown on SIGTERM/SIGINT

## Field Name Gotchas
- `TongueReading.CoatingColor` (not `Coating`)
- `PulseReading` has `Type` and `Characteristics`
- `Evidence` has `Content` (not `Description`)
- `Contradiction` has `Reason` (not `Description`)

## Testing Strategy
- Middleware tests: `httptest.NewRequest`, `gin.TestMode`
- Handler tests: Use `setupTestServer()` helper
- Integration tests: Test full diagnostic flow with real knowledge base
- Target coverage: ≥70%

## Dependencies
- `github.com/gin-gonic/gin` - Web framework
- `github.com/google/uuid` - Session IDs
- `github.com/stretchr/testify` - Testing assertions

## LM Studio Integration (Phase 4) — IMPLEMENTED
- **Package**: `internal/llm/` — provider-agnostic `LLMClient` interface + `LMStudioClient` (OpenAI-compatible, stdlib `net/http`) + `FakeClient` for tests.
- **What it does**: resolves *tied* formula candidates in step 12 (选方定药). When rule-based scoring can't decide (e.g. the 承气汤 family for 阳明腑实), the agent asks the LLM to pick, validates the choice is among the tied set, and records the reason in `DiagnosticSession.LLMRefinementReason`. On any failure (no client, no tie, network error, unparseable/invalid response) it falls back silently to the rule-based pick.
- **Config** (env vars, opt-in): `ONTCM_LLM_ENABLED=1`, `ONTCM_LLM_ENDPOINT` (default `http://192.168.50.17:1234`), `ONTCM_LLM_MODEL` (default `shizhengpt-7b-vl-i1`), `ONTCM_LLM_TIMEOUT` (default `60s`). Disabled by default — the server runs pure rule-based unless enabled.
- **Tests**: offline unit tests use `FakeClient` (refinement picks LLM choice; fallback on nil/error; LLM not called when no tie; invalid choice rejected). Live test gated behind `ONTCM_LLM_LIVE=1`.
- **Future use cases** (reuse the same client): dynamic question generation, free-text symptom intake, contradiction reasoning.

## Development Workflow
1. Make changes
2. Run tests: `go test ./... -v`
3. Build server: `go build -o /tmp/ontcm-server ./cmd/server`
4. Test manually: `curl` endpoints
5. Commit with proper message

## Key Files
- `cmd/server/main.go` - Entry point, graceful shutdown
- `internal/web/server.go` - Router setup, middleware chain, UI routes
- `internal/web/ui/` - Embedded SPA (`embed.go` + `static/` HTML/CSS/JS)
- `internal/agent/diagnostic_agent.go` - State machine
- `internal/agent/step_executor.go` - Step logic
- `internal/agent/question_templates.go` - Templates for each step
- `internal/knowledge/loader.go` - Load 112 formulas + 54 herbs
- `internal/knowledge/indexer.go` - Symptom→Formula mapping

## API Endpoints
```
GET    /                               - Web UI (embedded SPA)
GET    /static/*                       - UI assets (JS/CSS)
GET    /api                            - API info (JSON)
POST   /api/v1/diagnostic              - Start session
POST   /api/v1/diagnostic/:id/step      - Process step
GET    /api/v1/diagnostic/:id/state     - Get state
DELETE /api/v1/diagnostic/:id           - End session
POST   /api/v1/diagnostic/quick-formula - Quick recommendation
GET    /api/v1/formulas                 - List formulas
GET    /api/v1/formulas/:id             - Get formula
GET    /api/v1/formulas/search          - Search by symptom
GET    /api/v1/herbs                    - List herbs
GET    /api/v1/herbs/:id                - Get herb
GET    /api/v1/health                   - Health check
```

## Phase 7 Status (Web Interface) — COMPLETE
- Embedded SPA served by the existing Gin server (`internal/web/ui/`, `//go:embed all:static`). UI at `GET /`, assets at `/static/*`, JSON API info moved to `/api`. Vanilla JS + ES modules, CJK system fonts (offline), no new deps.
- **辨证 wizard** (`diagnostic.js`): renders each step's self-describing `question` payload by `type` (text/number/select/multiselect/textarea), drives the session loop, auto-advances reasoning steps 6→12, shows a 12-step stepper, the emergency-halt screen, and the final prescription (incl. `llm_refinement_reason`).
- **方剂 / 中药 lookups** (`lookup.js`): master-detail with instant client-side name/keyword filtering (finds a formula by name — e.g. "理中" → 理中汤 — which the symptom-only `/formulas/search` cannot).
- Verified: 太阴 case → 理中汤 end-to-end; emergency halt (`status=halted`); `/static` MIME correct (`text/javascript` for ES modules); `go test ./...` green (3 new ui tests).

## Domain Terminology
- 六经辨证 (Six Meridians diagnosis) - Core diagnostic method
- 方剂 - Herbal prescriptions
- 药证 - Herb-symptom matching
- 方证 - Formula-symptom matching
- 八纲 - Eight Principles diagnostic framework
- 十问 - Ten-question inquiry method

## Performance Targets
- Formula/herb lookup: <50ms
- Single step processing: <500ms
- Full 12-step diagnostic: <5s
- Concurrent sessions: 100+

## Phase 8 Status (Testing & Validation) — COMPLETE
- E2E test drives the full 12-step 太阴 case → reaches 理中汤 (`internal/agent/diagnostic_integration_test.go`)
- Accuracy benchmark across 5 canonical cases: 定经 5/5, 方证 5/5 (family-aware) (`diagnostic_accuracy_test.go`)
- Coverage: agent 83.5%, knowledge 77.8% (both exceed ≥70% target)

### Bugs found & fixed in Phase 8
- **Indexer keyword extraction** was byte-slicing Chinese (`text[i:i+2]`), producing invalid-UTF8 fragments that matched ~33 formulas per query. Now rune-safe, whole-term + delimiter split, plus index-side rune bigrams (patient "便秘" → formula "大便秘结多日").
- **ClinicalSign not indexed** — only the canonical symptom Name was. Now both Name and ClinicalSign are indexed, bridging formula↔patient vocabulary.
- **Template/step misalignment** — `GetStepTemplate(4)` returned the pulse template (step 5's), because var names didn't account for step 2 being the emergency gate. Renamed: `Step1Template`, `Step3Categories` (十问), `Step4Template` (舌诊), `Step5Template` (脉诊).
- **`contains()` helper was a no-op** in `handlers/diagnostic.go`, making quick-formula match everything. Replaced with `strings.Contains`.
- **Emergency triage was dead code** — `executeStep2` never runs (1→3 progression). Added a real emergency gate to step 1 that halts the session.
- **Step 7 scoring** now adds bonuses for matching the determined meridian and required symptoms, breaking ties (e.g. 麻黄汤 vs 桂枝汤 both matched 无汗).
- **`inferMeridianFromTongue`** classified the normal 淡红 tongue as 阳明 (naive 红 match). 淡红 is now correctly normal.
- Test path bug: `handlers_test.go` used `../../docs` (resolved to nonexistent `internal/docs`, silently loading 0 formulas). Fixed to `../../../docs`.

## Loader data-quality fixes (post-Phase-8) — DONE
- **Formula `Name == ID` (89/112 formulas)**: root cause was the markdown parser had no `# ` (H1) branch, so `Document.Title` was never set and name extraction fell through to a partial `formulaIDToChinese` map. Fixed: parser now captures the first H1 into `doc.Title`; loader derives the name from it (strips `药证详解`). All formulas now carry their Chinese name.
- **Herb overview columns mis-aligned** (桂枝 `Nature: "70"`, `Effect: ["心肺膀胱"]`): `loadHerbOverviewFile` read fixed positions, but tier1's table has an extra `出现次数` column tier2/3 lack → every field shifted left. Fixed: header-driven extraction (`herbColIndex`/`herbCell`) resolves columns by name; also populates `Frequency` and splits `核心药证` into `Effect`.
- **`index.md` loaded as a fake formula** (`{ID:index}`): `loadFormulas` now skips `index.md`. True unique-formula count is **111** (the prior 112 was inflated by the spurious index entry). `TestLoadAll` and the `/api` stats updated accordingly (stats are now dynamic from `loader.Stats()`).
- Regression tests: `pkg/markdown` H1 capture; `internal/knowledge` formula names, no-index, herb-column alignment.
- **`parseMeridians` left `MainMeridians` empty for all herbs**: `归经` cells concatenate organ names with no delimiter (`心肺膀胱`, `脾胃大肠`), but the old code only split on explicit delimiters → one unmatched part → empty. The organ→六经 table was also wrong (肺→太阳, 膀胱→少阴, 小肠→阳明, 肝→少阳). Fixed: longest-match token scan over a corrected table (太阳=膀胱,小肠; 阳明=胃,大肠; 少阳=胆,三焦; 太阴=脾,肺; 少阴=肾,心; 厥阴=肝,心包), dedup preserving order. All 54 herbs now have non-empty `MainMeridians` (桂枝→少阴,太阴,太阳). Regression test: `TestParseMeridians`.
- **桂枝加大黄汤 duplicated across `taiyin/` + `other/`**: a 34-line stub in `taiyin/` and an 85-line full doc in `other/`. The loader's dir order (taiyin before other) made the `other/` copy win the shared ID, mis-classifying this 太阴 formula (原文 279条 "属太阴也") as `MeridianOther`. Fixed: consolidated the full doc into `taiyin/`, deleted the `other/` copy, removed the stale nav row from `other/index.md`. 太阴 formula count went 6→7. Regression test: `TestGuizhiJiaDahuangConsolidated`.
- **Three filename-spelling duplicates** (same formula under two IDs → inflated count): 桂枝加芍药汤 (`guizhi_jia_shaoyao_tang` taiyin/ vs `guizhi_jia_shao_yao_tang` other/), 半夏散及汤 (`banxia_san_ji_tang` vs `banxia_san_tang`, both shaoyin/), 茯苓桂枝甘草大枣汤 (`linggui_gancao_dazao_tang` vs `linggui_ganzao_dazao_tang` typo, both taiyang/). Unlike the 桂枝加大黄汤 case the two filenames differed, so both loaded as separate IDs. Each consolidated to the canonical ID — `shaoyao` (single-token herb name, repo convention), `banxia_san_ji_tang` (captures 及, already test-referenced), `gancao` (correct pinyin vs the `ganzao` typo) — keeping the fuller content and deleting the stub/typo file. 桂枝加芍药汤's keeper also reclassified 其他→太阴. Unique count 111→108. Regression test: `TestFilenameDuplicateConsolidation`.
- **Variant table formats now parsed (Composition + OriginalText)**: `ExtractFormula` was 4-col-only (`药味|剂量|功效|归经`), so ~49 formulas with 3-col `药味|用量|功效` or 2-col `药味|功效` tables loaded with empty `Composition`. Rewritten header-driven: resolves 药味/用量|剂量/功效/归经 by name (归经 and dose optional), tolerates 2-5 cols, and skips repeated header rows (aggregate docs like 承气汤类 merge sub-tables). `Formula.OriginalText` was empty for ALL formulas — `GetSection("《伤寒论》原文")` is an exact map lookup that misses the `一、《伤寒论》原文` title; switched to substring match. Result: 79/108 formulas now have Composition (was ~30), 104/108 have OriginalText (was 0). Safe: Composition/OriginalText aren't used in scoring. Regression tests: `TestExtractFormulaThreeColumn`, `TestExtractFormulaTwoColumn`, `TestExtractFormulaSkipsRepeatedHeaderRow`, `TestFormulaOriginalTextLoaded`.
- **Step-12 tie-breaking made deterministic**: tied `FormulaCandidates` kept their map-iteration input order under the old `>`-only selection sort, so the 承气汤 family flipped run-to-run (the accuracy test's informational `方证 exact` oscillated 4/5↔5/5). Added `candidateLess` — a total order: MatchScore desc, then specificity (fewer total 方证要点 ranks first, so aggregate overviews like 承气汤类 stop winning ties), then FormulaID. The 阳明腑实 case now stably picks `xiao_chengqi_tang` (a real formula) instead of the 承气汤类 aggregate. The remaining exact-match gap (小承气 vs the severity-correct 大承气汤) is clinical severity = LLM territory. Regression test: `TestCandidateLessTiebreak`.
- **Drug-syndrome (药证校验) data loaded for both schemas — step 8 now activates**: two table variants existed and neither fed step 8. Schema A (`功效|对应症状|校验要点`, one table per herb) loaded but with `HerbName=""` for every row (the herb name lives in the preceding `### 药味——功效` heading, which the loader never read), and the parser merges the per-herb tables into one blob (blank lines / `---` are filtered before table parsing), so repeated header rows leaked in as garbage `DrugSyndromes`. Schema B (`药味|对应症状|作用机制`, herb in col 0) was rejected outright by a `Headers[0]=="功效"` gate, so `DrugSyndromes` was empty. Fixed: `ExtractDrugSyndrome` is now header-driven (resolves 药味/功效|作用机制/对应症状/校验要点 by name, skips seam rows, maps 作用机制→Effect); new `drug_syndrome.go` splits the merged Schema-A table on seam rows and pairs each group with its `### herb` heading (strips `（…）` annotations, splits `、`, leaves concatenated multi-herb names unsplittable). Result: **46 formulas carry DrugSyndromes, all with real HerbNames** (was ~22 with `HerbName=""` + 24 empty). Step 8 now fires: e.g. 麻黄汤 5.10 vs 桂枝汤 4.80 — the +0.10 is 桂枝's bare-`无汗` syndrome matching the patient's 无汗. Accuracy re-validated: 5/5 meridian, 5/5 family-aware (unchanged — picks robust to the small +0.1/herb nudges), deterministic over 5 runs. Regression tests: `TestExtractDrugSyndromeHerbDriven`, `TestExtractDrugSyndromeSkipsSeamRows`, `TestDrugSyndromeSchemaA`, `TestDrugSyndromeSchemaB`.
- **Step-8 matching made term-level + per-herb; aggregate overview excluded**: with DrugSyndromes populated (above), step 8's old match `strings.Contains(symptom.Symptom, syndrome.TargetSymptom)` still fired almost never — it required the patient string to contain the drug's *entire* `、`-joined phrase (`乏力、少气懒言`), while patient symptoms are `"<label>: <value>"`. Fixed: `drugMatchesAnySymptom` strips `（…）` annotations, splits the target on `、/，/,`, and accepts a term-level substring match against any patient symptom. The counting was also per-syndrome (a herb with several matching syndromes was counted several times); now per-herb (+0.1 per herb with evidence, max once). Step 8 now contributes broadly (e.g. 麻黄汤 went 5.10→5.20 as more of its herbs' terms match). Side effect caught and fixed: the now-active step 8 over-rewarded the 承气汤类 aggregate overview (it indexes three formulas' herbs/symptoms, so it out-scored the real formulas it summarized, defeating the `candidateLess` specificity tiebreaker). Resolved by excluding `X类` aggregate docs in the loader — each 承气 member already has its own dedicated doc (`tiaowei_chengqi_tang.md` IS 调胃承气汤, so nothing is lost). 阳明腑实 again picks the real `xiao_chengqi_tang`. Formula count 108→107. Accuracy re-validated: 5/5 meridian, 5/5 family-aware, deterministic (5 runs). Regression tests: `TestDrugMatchesAnySymptom`, `TestAggregateOverviewExcluded`.
- **Step-8 herb-name normalization — processed herbs now score**: step 8's herb↔syndrome join was an exact `syndrome.HerbName != herb.Name`, so a processed herb never matched its own 药证 entry — the two sources record the processing differently (composition writes it in parens `甘草（炙）`; the drug-syndrome heading uses a prefix `炙甘草`). Across 16 formulas with DrugSyndromes, this left 炙甘草/伏龙肝/桂枝（重用）/芍药（倍量） silently unscored. Fixed: new `normalizeHerbName` (strips `（…）`/`(...)`, then any leading 炙/酒/炒/煅/醋 processing prefix, looping so `酒炒` clears) compared via new `herbMatches`; step 8 now joins on normalized forms. 生/干 are deliberately NOT stripped — `生姜` and `干姜` are distinct herbs, not processed forms of a shared base, so conflating them would be a clinical error. Bridges real gaps (e.g. 理中汤's 甘草（炙）↔炙甘草); supporting herbs with no 药证 entry at all (甘草/生姜/大枣 in some docs) remain unscored — that's missing data, not a name mismatch. Accuracy re-validated: 5/5 meridian, 5/5 family-aware, deterministic (5 runs) — the added +0.1/herb weight didn't change any winner. Regression tests: `TestNormalizeHerbName`, `TestHerbMatches`.


## Next Steps
All 8 phases are complete. Candidate follow-ups (data quality / enhancement):
- **Patient↔drug vocabulary bridge (residual)**: step-8 term-level matching + herb-name normalization are in (see data-quality section above), but a herb still only scores when a drug's `TargetSymptom` *term* literally appears in a patient symptom string. Patient↔drug wording gaps remain — e.g. patient `稀软` vs drug `大便稀`, or patient label phrasing the drug doesn't echo. A synonym/alias table mapping common patient expressions to canonical drug-symptom terms would close these. Behavior change — re-validate accuracy.


- **承气汤类 aggregate doc** — RESOLVED: `yangming/chengqi_tang.md` (a 大/小/调胃承气汤 comparison overview whose merged sub-tables polluted the 阳明 candidate set) is now excluded from the loader as a non-formula. No data lost — `tiaowei_chengqi_tang.md` is a complete standalone 调胃承气汤. (The earlier note that 调胃承气汤 "exists only as a sub-section" was stale.) The file stays on disk as reference. The 阳明腑实 exact-match gap (小承气 vs severity-correct 大承气汤) remains clinical severity = LLM territory.
- `other/index.md`'s bottom stat table (其他类 24首 / 总计 41首) doesn't match its actual row count — the index is a curated subset, not a full listing. Reconcile if it's meant to be authoritative.
- Expand the synonym/vocabulary bridge for patient↔formula terms (currently relies on ClinicalSign + bigrams).
- Optional: dynamic question generation / free-text symptom intake via the LLM client (Phase 4 future use cases).