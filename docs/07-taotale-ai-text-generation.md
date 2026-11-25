# 07 — TaoTale: AI text generation (post-MVP feature specification)

**Status: DEFERRED — NOT PART OF THE CURRENT MVP.** This document preserves reusable design decisions and defines a *proposed* future direction. It is **not authorization to implement AI endpoints, screens, credential flows or paid calls** during the current MVP.

**Authority:** [01 — Product overview](01-taotale-overview-and-features.md) determines scope; [03 — Client](03-taotale-client.md) defines SolidJS architecture; [04 — Backend](04-taotale-backend.md) defines Go/Gin/provider adapters; [05 — Database](05-taotale-database.md) defines the current local PostgreSQL model; [06 — Segmentation and audio](06-taotale-segmentation-and-audio.md) defines the existing Azure TTS / Reader pipeline. The old Mandarin Reader AIT-01–15 and D-56–61 are **historical design input**, not automatically approved implementation or authorization methods.

## 1. Feature boundary and objective

A future **AI Text/Audio Generation Workspace** may help the learner create interesting reading material in a chosen learning language, optionally integrating already-saved vocabulary, then consciously review and save it using the standard Library lifecycle.

- **Current MVP:** The user manually creates/edits a Library item, explicitly generates **Azure TTS audio**, publishes a verified recording, reads/listens and collects vocabulary. **No AI text generation is needed.**
- **Post-MVP:** User selects a language, enters instructions and optional constraints, explicitly requests a text candidate, reviews/edits/refines it, then explicitly saves a Draft or enters the existing TTS/publishing flow.
- **AI is not required to synthesize the audio.** “AI Text/Audio Generation Workspace” means an optional workspace that can orchestrate **AI-authored text + existing Azure TTS**; it does not introduce a second, independent audio asset/publishing system or require an AI audio model.
- **No automatic writes or purchases:** Generating a candidate must never silently save it, create vocabulary, create Topics, submit a TTS request, publish, or retry an uncertain billable call.
- **No standalone AI account for TaoTale:** TaoTale remains a **single-user local app with one internal User**, no sign-in/out/onboarding/billing. Any external AI provider authorization is a distinct provider concern, not an application identity or a new multiuser system.

**Excluded from this planned feature unless separately decided:** vocabulary practice, language islands, generated comprehension quizzes, AI grading, automatic difficulty scoring, automatic Acquired status, AI conversation, study roadmap, full generated-article history, public sharing, cloud data synchronization, phone support, autonomous content generation and offline local inference. They are not implicit dependencies.

## 2. Cross-language product model

**Every generated article targets exactly one user-selected learning language.** MVP languages are `zh-CN` (Simplified Mandarin) and `ko-KR` (Korean). When this later feature is built, it should support languages enabled in the TaoTale catalog; do **not** infer language from the prompt or route a Korean request through a Mandarin-only model validator. Japanese (`ja-JP`) remains a future language and needs its own validated feature support.

| Concept | Proposed behavior |
|---|---|
| Language | Required BCP-47 supported learning language. Included in each AI request and candidate; preserved when saving. |
| Difficulty | Optional **Beginner / Intermediate / Advanced** guidance only. **Native is removed**; `null` means Unspecified. Never auto-assign the Library item's difficulty from generated text. |
| Language proficiency scale | **No required HSK/TOPIK field** in the generic feature. An optional HSK-specific control for Mandarin or TOPIK-related guidance for Korean would require separate per-language design and labeling. No proficiency certification claims. |
| Description/prompt | User-provided theme, scenario or instructions; treated as untrusted external-model input. |
| Target length | User-adjustable within the **current backend article/voice limits**. Historical 100–1500 UTF-16 target and ±10% guidance are *reference values*, not permanently approved cross-language limits. |
| Style | Optional suggestions, e.g. informational, story, dialogue, historical narrative, freeform. Style options are not part of the shared Library schema. |
| Required words | Optional manually entered phrases and/or selected **same-language** `Learning`/`Acquired` entries. |
| Optional words | Same-language vocabulary that the model may include when it fits. |
| Topics | Existing globally shared Topic IDs may be suggested/applied with explicit user approval. No automatic Topic creation. |
| Generated result | A separate, editable **candidate** comprising title, optional description, exact body and optional existing Topic suggestions. Not a Library draft by itself. |

**Multilingual input rules:** The prompt can be in the user's UI language (English in the MVP), while the generated article body must use the selected learning language. Chinese Simplified-script guidance is not a generic “non-Latin script” rule; Korean validation must not depend on Han characters. Do not silently convert scripts, infer a Korean lemma, translate vocabulary, or claim a model reliably meets a difficulty level.

## 3. Proposed workspace flow (future; no MVP route)

The workspace may eventually be reached from Library via a new route such as `/generate`, **but that route is not in the MVP client or API**.

1. **Choose language.** Default may follow the current active language; the request pins its explicit `languageCode`. Switching language clears or requires confirmation for vocabulary selections and unsubmitted candidate state rather than silently mixing languages.
2. **Describe content.** Enter a topic/instructions, approximate length, optional style, difficulty guidance and optional saved vocabulary constraints. Show clear indication of which data will be sent to the provider.
3. **Select vocabulary deliberately.** Add words manually, select saved same-language entries or ask for an explicit bounded sample. Keep selected IDs stable until the user changes or resamples them. No hidden background query that changes selections during a generation.
4. **Generate on explicit action.** The server validates request and provider readiness; user sees pending/failed/success states. A successful result is a **new candidate**, not a committed item.
5. **Review and edit.** Render text safely as plain text, show input/output length and required-word coverage. Let the user **Use candidate** or **Keep current** rather than destroying edits with a late response.
6. **Refine on explicit action.** A refinement request includes the exact **currently edited** candidate and instruction; its result is independently reviewable and never silently overwrites the current text.
7. **Save as Draft** through the existing Library creation endpoint; or **Generate Audio / Publish** through the existing Azure TTS preview and validated publication flow. Reuse Topic IDs, Favorite metadata, per-language voice defaults and manual difficulty. No special AI-published item subtype is required.

The future workspace must use the shared Create/Edit components and publishing behavior where practical. A generated article is an ordinary Library item **only after explicit save/publish**. No generated-candidate table, island table, separate audio jobs or persistent AI chat transcript is needed by default.

### 3.1 Unsaved state and navigation

- Before explicit Save Draft, edited AI candidate text and form selections live in **SolidJS local reactive state**; no automatic draft or server-side article state.
- Use the same dirty-navigation semantics as the manual editor: Save Draft & Leave (when valid), Leave Without Saving or Stay, plus native `beforeunload` warning for browser close/reload. Never claim unsaved browser work is recoverable.
- Pending or unknown-result billable generation and publication must preserve the immutable request intent and prevent an accidental second submission; it should **not** freeze unrelated reading or permanently trap navigation.
- If the user saves as Draft after reviewing a generated text, the Draft is the source of truth. Future edits to text, voice, rate or segmentation identity must trigger the same audio-outdated rules as any manual article.

## 4. Vocabulary inclusion and validation (future)

The old Mandarin Reader AI flow supported manually supplied Required/Optional word lists and deterministic sampling from Learning/Acquired vocabulary. Retain the **concept**, but do not blindly copy the old Chinese-only HSK, whitespace tokenization or raw phrase-match assumptions to Korean.

**Proposed safe behavior:**

- Select vocabulary from `(userId, selectedLanguageCode)` only; reject mismatched/deleted/foreign IDs even in the single-user app. Topics are globally shared but word associations are language-scoped.
- Treat **Required** words as strict text-coverage constraints and **Optional** words as suggestions. If the same normalized key is present in both, Required takes precedence.
- Keep one canonical, versioned normalization policy for **request deduplication** without changing the stored word's spelling. Language-specific case/spacing rules may require adjustment; do not infer Korean dictionary forms.
- Manual phrases need not already exist in Vocabulary. Model-generated meanings, readings or status changes must **never be silently written back**.
- A server-side Select/Reshuffle action may sample eligible saved entries deterministically from a bounded dataset, returning the **actual** count and handling insufficient availability. Preserve selected IDs until explicit reselection. No per-generation Library-wide background vocabulary scan.
- On each Generate/Refine, verify selected IDs and status/language again. Cap vocabulary count and prompt size to avoid impossible constraints and unexpected provider costs. The old limits (up to 25 Required / 40 Optional effective; 0–20/0–40 additional sampled) are **historical starting points** to re-evaluate per language and article-length cap.
- Verify required words against the returned exact body using a documented per-language *surface-form* rule; make clear that literal substring coverage **does not prove grammatical or meaningful use**. Korean morphology and particles make exact surface-form inclusion especially limiting. Optional words and difficulty/length targets are advisory unless independently validated.
- If required-word coverage fails, return a structured validation error showing missing phrases. **Do not silently rewrite, autocorrect or retry a paid generation** to force compliance.

**Separation of concerns:** AI vocabulary inclusion is a prompt/validation feature. It does **not** replace the per-language speech segmenter, existing manual vocabulary linking or any independently tested language-aware occurrence matcher from 06/05.

## 5. Output contract and safety (future)

A versioned response contract should yield, at minimum:

```json
{
  "languageCode": "ko-KR",
  "title": "한국어를 공부하는 하루",
  "description": "A short story about a study routine.",
  "text": "저는 매일 한국어를 공부해요.",
  "suggestedTopicIds": []
}
```

*Illustrative, not a guaranteed model response or an approved OpenAPI schema.*

Server requirements when implemented:

- Construct a trusted, versioned system instruction with untrusted JSON user data separated from the instruction layer. Library text, stored words, Topics and refinements are **data**, never elevated into system instructions.
- Request structured output if supported by the **selected and authorized** AI provider, then independently validate the final parsed response. Handle incomplete streams, invalid JSON, missing keys, oversize output and unsupported language. Keep request/response byte limits explicitly configured and tested.
- Ensure returned `languageCode` matches the requested language (or derive it only from the immutable server request and ignore untrusted model-provided mismatch). Language/script checks are **guardrails**, not proof of grammatical correctness or proficiency level.
- Validate title/body/description against the **active TaoTale Library limits** (see 01), not the old Chinese-only fixed schemas. Preserve the exact selected candidate text; never silently trim a generated body to fit TTS.
- Accept suggested Topic IDs only if they exist in the local user's catalog and respect association caps; unknown suggestions are discarded with an understandable notice, **not newly created**.
- Validate required-word coverage, report optional-word usage and length deviation honestly. Never claim CEFR, HSK, TOPIK or manually assigned difficulty has been objectively certified by the model.
- Render candidate as escaped plain text; never execute generated markup, script, external links or component code as UI. Review text before any Azure submission.
- Do not log full prompts, articles, vocabulary, secrets, provider token contents or raw error bodies; log metadata-only request IDs, durations and sanitized error codes.

**Initial policy:** No model-driven retry, self-correction loop, automatic article save, unreviewed publishing, comprehension questions or full conversation history. Those features would need separate scope approval.

## 6. Go backend adapter and data boundaries (future)

The current **Go + Gin + pgx/v5 + sqlc** backend remains the authoritative local application. The AI feature, when prioritized, should add a small, isolated capability (e.g. `internal/aitext`) rather than reintroducing NestJS modules, Prisma ORM or a second service/database.

Suggested provider-neutral contract:

```go
// Illustrative only: implement when the post-MVP feature is approved.
type TextProvider interface {
    AvailableModels(ctx context.Context) ([]Model, error)
    Generate(ctx context.Context, req TextRequest) (TextResult, error)
}
```

- Retain the **previously chosen AI provider** behind an adapter. Its actual account/credential mechanism, model list, supported output format, token budgets and terms must be **revalidated when implementation begins**. Do not assume the former Windows/Node/ChatGPT connection mechanism can be transplanted to Go or is currently authorized.
- The provider must not import or write Library, Vocabulary or Topic domain entities. The orchestration service **reads** selected owned words/Topics and returns candidates; ordinary item endpoints perform explicit saves and publishing later.
- The app's **local internal User** is always the app owner. An AI provider's external account, subscription or credentials do **not** create/sign into a TaoTale user, and an AI provider connection is not authentication for local item ownership.
- Do not create AI connection, history, model-cache or generated-article tables in the MVP schema. A bounded local idempotency/replay record may later hold **sanitized** results or claims if needed; the old 24-hour policy is a historical starting point, not a current schema mandate.
- Keep provider secrets in local backend-only configuration or an **approved OS-specific credential store** with restricted file permissions. Never store plaintext provider tokens in PostgreSQL, media directories, frontend `localStorage`, Git or client bundles. No blanket claim of secure support for unverified OS/runtime combinations.
- Gin applies the existing loopback-only deployment and strict origin/CSRF protections for sensitive actions. Being local-only is **not** sufficient to trust every browser-originated mutation. If remote LAN access is later enabled, revisit the trust boundary explicitly.

### 6.1 Request lifecycle and idempotency

When AI generation is eventually introduced, distinguish it from ordinary local CRUD:

1. Validate request and provider availability **before** committing an external attempt.
2. Record a stable client-generated idempotency key, immutable operation/request hash, model/provider identity and a conservative attempt claim **before** the external request when necessary to avoid duplicate paid calls.
3. Execute model inference **outside PostgreSQL transactions**, with a configured deadline and output cap. Never retry a timeout, dropped connection or uncertain provider outcome automatically.
4. Persist a sanitized terminal outcome (or an explicit interrupted/unknown state) as the chosen replay design requires, allowing same-key/same-intent recovery and rejecting reused keys with changed input.
5. A fresh attempt after confirmed failure requires an explicit user action and **new key**. At-most-once local submission intent is attainable; *exactly-once provider execution or guaranteed recovery of a lost model response is not promised*.

The former D-58 committed-claim/24-hour scheme is a useful candidate, **not an adopted Go implementation**. Do not allocate AI worker capacity, schedule background AI tasks or initialize provider credentials during current MVP startup. No separate audio generation service is created: Azure TTS already has its own durable job lifecycle.

## 7. Optional voice and publishing handoff (future)

The workspace eventually delegates to the existing `LibraryItem` / `GenerationJob` / `AudioAsset` workflow:

- A candidate includes a pinned `languageCode` and user-edited exact text. The user must select a voice compatible with that language and a provider-supported synthesis rate.
- **Generate Audio** is a separate, explicit Azure TTS submission, using the existing generation endpoint, immutable fingerprint, per-language strategy and validated segments from 06. AI text does not make audio automatically `READY`.
- A temporary audio preview belongs to the exact candidate text + language + Azure provider/voice/rate + strategy/version. Modifying synthesis-affecting inputs invalidates the preview; editing title, description, Topics, Favorite or manual difficulty does not.
- Explicit **Save as Draft** uses the existing draft API. Whether a matching generated preview can be claimed follows the ordinary draft/preview rules, not a special AI-only shortcut; the historical choice to leave an AI preview unclaimed can be reconsidered only when implementing the workspace.
- Explicit **Publish** requires a saved or direct-publish transaction with a verified matching READY asset. The existing Language/Vocabulary reference integrity, media lifecycle and publication concurrency rules remain in force. No AI shortcut may bypass audio alignment or generate entries in an untested language.
- On failure/unknown outcome, keep the candidate and original immutable intent available for recovery; never publish a mismatched preview or silently submit a second billable request.

## 8. UX and performance expectations (future SolidJS client)

- If prioritized after MVP, build the workspace in the **existing TaoTale SolidJS + TypeScript + Vite + Solid Router client**, using **app-owned Iris CSS Modules and native controls** from [08](08-taotale-design-system.md). Do **not** add a separate UI library, Kobalte/headless system or generic component-behavior framework.
- Load the AI route and any optional provider UI **lazily** when introduced. The MVP bundle and startup should not contain a required AI connection or model-catalog fetch.
- Keep form edits, word selections, candidate comparison and errors in Solid primitives; avoid global mutable stores for a single workspace. Use native fetch plus the shared typed API contract. Avoid adding framework-sized form/query packages without demonstrated need.
- Show clear pending states, provider-unavailable/usage-limit errors, unknown outcome, explicit Retry, character limits, missing required words and a “data sent externally” disclosure before generation.
- Preserve the distinction between an AI candidate, a **saved Draft**, a **temporary audio preview** and a **published item**. They must never appear to be interchangeable saved objects.
- Keep the existing desktop/tablet support boundary; phone-specific workflow is post-MVP. Use simple labeled native controls and safe plain-text rendering consistent with [03](03-taotale-client.md) and [08](08-taotale-design-system.md), without specialized keyboard/focus/overlay JavaScript.

## 9. Privacy, reliability and cost controls

- **Local database and media do not mean local inference.** An external AI provider receives only data for the explicit action; the user should see which fields are sent (prompt, selected saved words, Topic labels, and, for refinement, the current article). Do not send unrelated Library history or all vocabulary automatically.
- Azure TTS is an **independent external request**, separately disclosed and triggered. No silent chained AI → TTS billing.
- Cap prompt size, selected vocabulary, output length, request concurrency and uncertain retries. Explain provider availability and usage limits as operational constraints, **not** TaoTale subscriptions/billing plans.
- Support an explicitly labeled mock/fake adapter in tests, with no actual paid calls and no confusion with connected provider results.
- Handle model hallucination, mistaken facts, wrong-language text and inappropriate style as reviewable output problems; do not represent the AI as a language proficiency assessor.
- Keep account secrets out of logs/backups. If an OS secret store is later used, document whether a full app backup requires a separate provider reconnection after restore.

## 10. Verification plan and release gates — only when this feature is scheduled

**Proposed later tests:**

1. Generating in `zh-CN` and `ko-KR` cannot mix vocabulary, voices or script checks; a language switch requires explicit reselection of incompatible constraints.
2. Required/Optional precedence, frozen sampled IDs, deleted-word races, duplicates, missing coverage and unachievable text-length goals produce honest validation or warnings.
3. Candidate Generate/Refine never writes a Library item, vocabulary entry, Topic, audio job or public history; late response never overwrites local edits without acceptance.
4. Save as Draft uses ordinary Go/sqlc item API; Azure voice preview and Publish use the **existing** verified audio/segmentation pipeline for the selected language.
5. Repeated/unknown billable requests with the same idempotency key do not create automatic duplicate attempts; model failures leave work intact and emit sanitized errors.
6. External-data disclosure, no secrets in client/logs/database, explicit provider connection when applicable, and tested local OS/runtime credential protection.
7. SolidJS route is lazy; errors, keyboard and focus flows meet the future desktop/tablet UX gates.

**The current MVP is complete without any of these AI tests, routes or provider credentials.** Its release gate is manual Library + Chinese/Korean Azure TTS + Reader + Vocabulary + Favorites + local PostgreSQL/media integrity/restore, as defined in 01/04/05/06.

## 11. Historical Mandarin Reader decisions and open future questions

| Historical area | TaoTale disposition |
|---|---|
| AIT-01–15 and D-56 (Chinese prompt/HSK/caps) | **Defer/adapt.** Preserve explicit actions, validation and sampling concepts; remove Chinese-only universal assumptions. |
| D-57/58 (Windows ChatGPT connection, DPAPI, SIWC, provider claims) | **Historical, not adopted as-is.** Reverify authorization, current provider interfaces, OS support and credential storage for a Go backend if still relevant. |
| D-59 (separate generation workspace and candidate review) | **Post-MVP design reference.** No `/generate` or candidate API in current MVP. |
| D-60 (Voice & Publish) | **Post-MVP UI reference.** Reuse existing Azure editor generation/preview/publish invariants instead of parallel paths. |
| D-61 (disclosure, limits, recovery and clear actions) | **Retain principles when feature is prioritized**; provider and protocol-specific implementation must be revalidated. |

**Questions reserved for future implementation planning (not MVP blockers):** exact retained AI provider/model/authorization approach; required word coverage semantics per language; future prompt/output cap; whether candidates should survive browser reload; provider data-retention settings; whether the user wants HSK/TOPIK as optional *advisory* inputs; AI preview claim behavior on Save Draft; how provider credentials are stored on the installed OS. Do not prebuild tables/routes/SDK dependencies to answer questions that have no current product requirement.

**Maintenance rule:** When this feature is explicitly brought into scope, update 01/02, define its OpenAPI endpoints, approve Go adapter and credential strategy, extend 03/04/05 only where needed, and add the feature to acceptance tests. Until then, this is a **design reference only**.
