# Mobile layout pass

Frontend-only changes, developed against a local synthetic timeline. No production
configuration, data, importer, image or deployment changes.

## Changes

- Conversations/Timeline use full-width phone columns rather than desktop 25/75.
- Native keyboard-accessible filter disclosures start closed below768px; desktop
  filters open automatically. Query changes retain the disclosure state.
- Conversations keep a bounded scroll pane; opening filters does not replace the
  message scroller. Long text and media stay inside the available width.
- Items retain timestamps on phones, reduce card side rails, and keep the View
  action visible with a44px touch target. Toolbar filters collapse separately from
  dates/people. Numeric page-size input has an accessible name/input mode.
- Mobile navigation collapses after choosing a destination. Header spacing and
  common control sizes are adjusted; the import icon has an accessible label.
- Gallery sidebar is collapsible, cards can shrink to phone width, preview content
  height is bounded, and desktop hover enlargement is disabled on phones.
- Dashboard heading no longer occupies a fixed two-column slot on phones.

## Local verification

Phone CSS widths320/360/412 and desktop1280 checked in the real local application
with synthetic Facebook posts and three80-message conversations. No document-wide
horizontal overflow in Items, Timeline, Gallery or Conversations. Visible item
metadata and44px View buttons checked. A conversation opens with50 messages;
scrolling to the beginning loads the remaining30, preserving the scroll pane.
Source filtering preserves the open panel. Resizing to desktop reopens filters.
Choosing a navigation destination closes the mobile menu. JS syntax and diff
checks pass.

Checks use browser DOM/layout and interactions, not merely static HTML. Screenshot
capture timed out; fallback browser access was unavailable. Native iOS/Android,
virtual-keyboard behavior and image/video-heavy conversation layouts still need
hands-on review. Gallery card behavior was inspected structurally; the local
fixture has no media payloads. This is a bounded layout improvement, not a full
mobile redesign or cross-browser certification.

Local operator demo: http://localhost:13802 (Mac loopback only), container
`timelinize-mobile-demo`, isolated anonymous volumes and synthetic inputs only.
Its frontend is copied from this checkout; refresh after copying updated assets:

```sh
DOCKER_HOST=unix://$HOME/.colima/agent/docker.sock docker cp frontend/. timelinize-mobile-demo:/ui
```

Stop/remove only the activity container and its anonymous demo state:

```sh
DOCKER_HOST=unix://$HOME/.colima/agent/docker.sock docker stop timelinize-mobile-demo
```

No release tag or production image is made by this local pass. AI-assisted changes.
