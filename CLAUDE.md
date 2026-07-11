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
- **Formulas**: 112 total in `docs/formulas/shanghanlun/`
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
- `internal/web/server.go` - Router setup, middleware chain
- `internal/agent/diagnostic_agent.go` - State machine
- `internal/agent/step_executor.go` - Step logic
- `internal/agent/question_templates.go` - Templates for each step
- `internal/knowledge/loader.go` - Load 112 formulas + 54 herbs
- `internal/knowledge/indexer.go` - Symptom→Formula mapping

## API Endpoints
```
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

## Next Steps
- Phase 4: LM Studio integration for question generation and severity-based formula selection (e.g. exact 承气汤 member)
- Phase 7: Web interface (optional)
- Expand the synonym/vocabulary bridge for patient↔formula terms (currently relies on ClinicalSign + bigrams)