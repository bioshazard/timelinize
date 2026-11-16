# bios Timelinize fork

Modified 2026-11-16. An unofficial AGPL-3.0-or-later downstream, based on
upstream `11ecc5af75a5b0e0250c6197eb213e2b2b7f7c7d` (2026-05-21).
Original authorship, copyright and license notices remain intact. New downstream
changes were AI-assisted. The UI footer offers this repository's source; tags
preserve the exact source of released versions.

## First downstream release: v0.0.29-bios.1

Development prerelease for **fresh projections**. It is not a declaration of full
Facebook export coverage or a tested migration path from older databases.

- Upstream main already repairs HTTPS log WebSocket selection (`wss:`) and
  nil-metadata entity merges; those fixes are inherited here.
- Twitter URL/media expansion uses validated UTF-16 source spans, replaces from
  right to left, then decodes HTML entities. Invalid/ambiguous spans retain the
  source short URL instead of slicing unrelated text. Retweets use nested entities.
- Facebook post/message text occurrences have deterministic archive-member/array
  locations. Repeated text, case differences and different threads can stay
  distinct when strict `intermediate_location` identity is configured.
- Post attachment text is retained. Message attachment time prefers sent time;
  distinct supplied media time remains in metadata.
- Multipart retrieval honors importer field policies only for an exact retrieval
  key match, including attachment sender/classification precedence.
- Missing identity hashes cannot match arbitrary tombstones. With strict
  occurrence identity, no-ID remembered content hashes include source and
  occurrence; deleting one exact repeat does not suppress another.
- Text identity comparisons are binary; user-facing search stays case-insensitive.
- Recent conversation lists honor source filters before pagination.
- Inherited HTML rendering, hash and build fixes below; incoming relationship
  index supports reverse relationship lookups.

## Adopted work and attribution

Original commits are cherry-picked with `-x` where adopted whole:

| Source | Adopted change |
| --- | --- |
| [Chris Rowe / upstream PR198](https://github.com/timelinize/timelinize/pull/198), `43b975875265c8df0f499161c2b7944ad32282da` | Pin Docker libvips to v8.18.4. |
| [Stuart Atkinson](https://github.com/StuartJAtkinson/timelinize/commit/3421010), `3421010` | Conversation hash sorts a copy instead of mutating participants. |
| [Stuart Atkinson](https://github.com/StuartJAtkinson/timelinize/commit/e5d8c27), `e5d8c27` | Render inline HTML using the actual text variable. |
| [Steven Horner fork](https://github.com/stevenhorner/timelinize/commit/67c7950), `67c7950` | Only the incoming-relationship schema index; broad search rewrite not adopted. |

Reviewed, not merged wholesale: Stuart Atkinson's interactive import UI, additional
importers/module rename; Steven Horner's query rewrite; Samuel Moede's deployment
workflow and unrelated importer changes. Their licenses permit reuse, but their
behavior needs its own acceptance fixtures. These forks remain valuable sources
for later narrowly reviewed improvements.

Additional build fix: [Samuel Moede fork / Claude](https://github.com/samuelmoede/timelinize/commit/5584cb4810f84d0f5c63f05657b253a94847e111),
`5584cb4810f84d0f5c63f05657b253a94847e111`, removes forced Meson fallback dependency
builds. Docker and Linux CI share `.libvips-version`; Docker builds run the Go
suite before producing a binary. Multi-platform publishing configures QEMU.

## Import contract and limits

Use an immutable, deterministically ordered archive. Configure at least
`timestamp`, `data`, `data_source_name`, `classification_name`, and
`intermediate_location` as strict item unique constraints. Native IDs remain
platform-provided; array locations are derived provenance, **not Facebook IDs**.
Reordering/repackaging an archive changes array occurrence identities; this is
not a cross-export deduplication algorithm. Reconcile generations before import.
Fresh rebuilds are required to benefit reliably from the new occurrence hashes;
old tombstones and already collapsed items are not repaired by an upgrade.

Media still coalesces by archive URI/retrieval key. Multiple message occurrences
reusing the same URI do not become independent media items; the last applicable
context can win. Missing senders, timezone/DST ambiguity, unsupported Facebook
categories and absent source fields remain unresolved. Successful import is not
a fidelity proof. No fake source IDs, timestamps or senders are introduced.

The legacy Facebook HTML converter lives in the separate
[facebook-archive-converter repository](https://github.com/bioshazard/facebook-archive-converter).
This application consumes its explicit Timelinize JSON compatibility output; it
contains no legacy HTML parser. Private operator reconciliation receipts remain
outside public source repositories.
No personal exports, credentials or private fixtures are included here.

## Validation

Requires Go at least the version in go.mod, a C toolchain and libvips development
headers. The source release is tested on macOS arm64 with Go1.26.2 and
libvips8.18.7. CI/Docker use the exact Go1.26.2 toolchain in `.go-version`;
the floating `stable` alias selected Go1.27.4 and broke existing dependency/test
compatibility, so it is no longer used. Docker uses pinned libvips8.18.4 and needs separate image validation.

```sh
go test ./...
go test -race ./timeline ./datasources/facebook ./datasources/twitter
go build -o /tmp/timelinize-bios .
```

Synthetic tests cover exact occurrences, binary identity, tombstones, scoped
content hashes, multipart policy boundaries, conversation filtering and Unicode
URL expansion. Native job integration fixtures also exercise the registered
Facebook importer through processing and SQLite: exact repeated/case-sensitive
text across threads, media-only messages, stored text/media bytes, attachment and
recipient edges, media sender/classification/time, owner/source-account merge,
scalar metadata into a nil-metadata owner, idempotent replay, remembered deletion
isolation, new identical/unrelated occurrences after deletion, and repository
reopen/integrity. These are small synthetic fixtures, not full source-ledger
verification of a real projection. Production rebuild/rollout and backup/restore acceptance are separate
steps; publishing a source tag does not deploy it. Full Go suite, targeted race tests and native
macOS arm64 compilation passed; independent spec/standards reviews found no
blocking findings. Container builds and real projection acceptance remain separate.

## Observed Twitter archive index profiles

Some observed archive entities use Unicode code-point indices rather than UTF-16
units. The importer accepts a UTF-16 span first, then a code-point span only when
it exactly equals the declared source URL token. Invalid/stale spans and
conflicting/overlapping replacements remain refused. This is an observed archive
compatibility profile, not a universal claim about the Twitter API. Synthetic
fixtures cover both profiles, supplementary emoji, combining accents, URL and
media entities. No private export fixture is published; IDs/times are unchanged.
