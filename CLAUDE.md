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
- **Formulas**: 108 unique in `docs/formulas/shanghanlun/` (108 source files, no duplicates — 桂枝加大黄汤 + three filename-spelling duplicates consolidated. `index.md` files are skipped as navigation, not formulas.)
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
- **Variant table formats now parsed (Composition + OriginalText)**: `ExtractFormula` was 4-col-only (`药味|剂量|功效|归经`), so ~49 formulas with 3-col `药味|用量|功效` or 2-col `药味|功效` tables loaded with empty `Composition`. Rewritten header-driven: resolves 药味/用量|剂量/功效/归经 by name (归经 and dose optional), tolerates 2-5 cols, and skips repeated header rows (aggregate docs like 承气汤类 merge sub-tables). `Formula.OriginalText` was empty for ALL formulas — `GetSection("《伤寒论》原文")` is an exact map lookup that misses the `一、《伤寒论》原文` title; switched to substring match. Result: 79/108 formulas now have Composition (was ~30), 104/108 have OriginalText (was 0). Safe: Composition/OriginalText aren't used in scoring (step 8 is a no-op — `DrugSyndrome.HerbName` is always `""`). Regression tests: `TestExtractFormulaThreeColumn`, `TestExtractFormulaTwoColumn`, `TestExtractFormulaSkipsRepeatedHeaderRow`, `TestFormulaOriginalTextLoaded`.
- **Step-12 tie-breaking made deterministic**: tied `FormulaCandidates` kept their map-iteration input order under the old `>`-only selection sort, so the 承气汤 family flipped run-to-run (the accuracy test's informational `方证 exact` oscillated 4/5↔5/5). Added `candidateLess` — a total order: MatchScore desc, then specificity (fewer total 方证要点 ranks first, so aggregate overviews like 承气汤类 stop winning ties), then FormulaID. The 阳明腑实 case now stably picks `xiao_chengqi_tang` (a real formula) instead of the 承气汤类 aggregate. The remaining exact-match gap (小承气 vs the severity-correct 大承气汤) is clinical severity = LLM territory. Regression test: `TestCandidateLessTiebreak`.

## Next Steps
All 8 phases are complete. Candidate follow-ups (data quality / enhancement):
- **Drug-syndrome `药味|对应症状|作用机制` schema not parsed**: 26 formulas use this schema (vs the 88 `功效|对应症状|校验要点` the loader accepts via `Headers[0]=="功效"`). The loader skips them, so `DrugSyndromes` is empty and step-8 herb-symptom verification stays a no-op. Fixing it would populate real `HerbName` values and ACTIVATE step-8 scoring (+0.1/herb-with-evidence) — a behavior change needing accuracy re-validation. (Composition + OriginalText table-variant issues are already fixed — see data-quality section above.)
- **承气汤类 aggregate doc** (`yangming/chengqi_tang.md`): an overview comparing 大/小/调胃承气汤 with three sub-tables the parser merges into one. Loaded as a single "formula" with 16 aggregate KeySymptoms, it pollutes the 阳明 candidate set (now deprioritized by the `candidateLess` specificity tiebreaker, but still indexed). 调胃承气汤 (`tiaochengqi_tang`) exists ONLY as a sub-section here, so it can't just be deleted — needs splitting into a dedicated doc. Splitting it (and giving 调胃承气汤 its own ID) would let the 阳明腑实 case reach exact-match.
- `other/index.md`'s bottom stat table (其他类 24首 / 总计 41首) doesn't match its actual row count — the index is a curated subset, not a full listing. Reconcile if it's meant to be authoritative.
- Expand the synonym/vocabulary bridge for patient↔formula terms (currently relies on ClinicalSign + bigrams).
- Optional: dynamic question generation / free-text symptom intake via the LLM client (Phase 4 future use cases).