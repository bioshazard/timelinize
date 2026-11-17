# Downstream beta candidate: existing workflows

AI-assisted implementation. Baseline: `9dcd26e`; no new feature families,
frontend dependencies, bundler or framework. Original exports remain canonical.
The fork is a browseable projection, not the sole backup.

## Completion contract

- Profile Items/Map open existing views filtered to that entity; no dead tabs.
- Search/paging/filter URLs survive reload and Back, including multiple people.
- Empty results replace stale results. Errors leave controls usable and provide retry.
- Phone layouts fit 320–430px; desktop remains usable. Controls need no hover.
- Existing settings save/cancel and tab navigation work; unfinished configuration
  is labelled, not a fake actionable control.
- Existing merge/import/job actions retain native semantics; test mutations only
  on synthetic data. No production import, merge, deletion or settings change.
- No importer/storage/schema changes in this UX pass. Run existing source-fidelity
  regression tests once, not repeated NAS byte verification or rebuilding exports.

## Audit

Live baseline: Items rendered 51 rows for a 50-row page; Map reached 1,235px on
390px; Settings reached 631px; Profile Overview/Items/Map were `#` placeholders.
Local first pass already repaired item pagination, thread reopen state, failed
attachment waits, compact cards, gallery preview controls and phone calendars.

Read-only live inspection covers dashboard, Items, Conversations, Entities,
Profile, Map, Settings, import planning, image-only Gallery and Timeline.
Media previews can generate derived thumbnails; no videos requested.
Screenshots currently time out; DOM geometry and actual click interactions are
available. Real iOS/Android keyboard behavior remains a separate qualification.

## Release boundary

Local LAN candidate first. No tag, production rollout or beta-completeness claim
until interaction, mobile/desktop, error and source-fidelity checks are recorded.
Known unsupported export categories remain qualified in FORK.md and import docs.

## Candidate evidence

All changes remain on `mobile-ux`; the LAN demo serves this checkout at
http://192.168.1.126:13802. Production retains its existing image/configuration.

| Check | Result |
| --- | --- |
| Node behavioral regressions | 25 passed |
| Frontend JS syntax + diff check | Passed |
| `go test ./...` | All packages passed; existing cached importer/fidelity results reused |
| 320px and 1280px route sweeps | Ten existing routes each: no horizontal page overflow or render-error panel |
| 390px phone interactions | Profile → filtered Items; conversation open/reload/back; empty selections; Settings reset/save |
| Query round trips | Two people restored; limit10/page2 gives rows11–20 after reload |
| Browser Back | Items → previous profile succeeds |
| Fault injection | Failed Map template leaves a visible error; Retry reaches the intended filtered Map |
| Job detail | Existing synthetic import config renders at phone width |
| Live baseline | Read-only core pages and image-only Gallery;100 cards, zero videos requested |

Standards review: no documented-standard breaches. Requirements/correctness
reviews identified six issues, all repaired with focused regressions: Retry
routing, page-size persistence, keyboard submit, none/Unknown type distinction,
stale entity-filter restoration and stale Map date presearch.

No new frontend dependency, importer/storage/schema change, NAS payload scan,
production mutation commands/configuration changes, image build, release tag or deployment. Empty type selection
uses `types=none` in UI URLs; Unknown remains `class=`. Native empty-class API
semantics are preserved: the frontend resolves explicit none locally, for Items
and both Conversation endpoints.

Remaining qualifications: real iOS/Android keyboard/media behavior; a fresh-image
packaging/rollout smoke test; live job/merge/import mutations (deliberately not run
against personal data). Screenshot capture remains unreliable, so browser claims
are DOM geometry and actual control interactions, not visual certification.
Unsupported export categories and existing projection coverage gaps remain outside
this UX completion pass. This is a **local beta candidate**, not a blanket claim
that every upstream alpha feature is complete.

## Why this application

[Timelinize](https://timelinize.com/docs/) explicitly remains alpha upstream, but
its entity-aware, multi-source personal timeline matches this task.
[HPI](https://github.com/karlicoss/HPI) is primarily a programmable data interface;
[Dawarich](https://dawarich.app/docs/features/imports/) focuses on location history.
Those are useful adjacent tools, not evidence of a more complete drop-in browser
for our Facebook/Twitter projection. Retain original exports independently.

## Existing-session upgrade correction

An existing browser session reproduced mixed assets: updated profile HTML but
cached old `entity.js`/SPA code, leaving new links unwired. First-party script and
stylesheet URLs now contain a SHA-256 fingerprint rendered using the existing
trusted-template include/Sprig mechanism. No build step, manual version bump or
cache purge required; unchanged content retains a stable URL. These trusted
first-party assets must not contain literal Go-template delimiters. Template
render tests cover the real application entrypoint and changed/unchanged bytes.

Retested `/entities/e92723a8-9ec9-49ce-ab2c-a41e04c92a6e/1` in the previously cached
session: Items and Map navigate with entity1 selected; Merge opens with entity1
as keeper. No merge was submitted. The synthetic owner currently has no owned
items, so its Items page correctly shows an explicit empty state. The imported
synthetic account is entity2; this is fixture identity separation, not missing
personal production data. Production remains unchanged.
