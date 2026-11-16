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

The legacy Facebook HTML conversion adapter and private reconciliation receipts
remain in biosnet, separate from this application fork:
[converter snapshot](https://gitlab.com/bios-dev/incubator/biosnet/-/tree/f02280f/tools/facebook-export-conversion).
No personal exports, credentials or private fixtures are included here.

## Validation

Requires Go at least the version in go.mod, a C toolchain and libvips development
headers. The first source release is tested on macOS arm64 with Go1.26.2 and
libvips8.18.7; Docker uses pinned libvips8.18.4 and needs separate image validation.

```sh
go test ./...
go test -race ./timeline ./datasources/facebook ./datasources/twitter
go build -o /tmp/timelinize-bios .
```

Synthetic tests cover exact occurrences, binary identity, tombstones, scoped
content hashes, multipart policy boundaries, conversation filtering and Unicode
URL expansion. They do not replace full source-ledger verification of a real
projection. Production rebuild/rollout and backup/restore acceptance are separate
steps; publishing a source tag does not deploy it.
