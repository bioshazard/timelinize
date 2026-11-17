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
