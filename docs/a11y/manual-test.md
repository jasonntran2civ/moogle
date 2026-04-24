# Manual accessibility checklist

axe-core catches a lot but not everything. Before each launch and after any UI change touching ARIA / focus / interactive widgets, run through this list.

## Screen reader

Use **NVDA on Windows** and **VoiceOver on macOS**.

- [ ] Tab through home page; every interactive element gets an announcement.
- [ ] Type a query; press Enter; results stream is announced via the `aria-live="polite"` region as each wave arrives.
- [ ] j/k navigates results without losing focus to the page.
- [ ] Open a result drawer with Enter; Esc closes; focus returns to the originating result.
- [ ] COI badge is announced with full content ("COI $147,000 from Manufacturer"); not just "button".

## Keyboard only

- [ ] No mouse needed for any flow: home → search → drawer → docs.
- [ ] Skip-nav link appears on first Tab and lands at `#main`.
- [ ] `/` focuses search input even mid-page.
- [ ] `?` opens the keyboard shortcut help (TODO: implement modal).
- [ ] `Esc` closes the most-recently-opened overlay.

## Color & contrast

- [ ] All text + non-text content meets WCAG 2.2 AA contrast (4.5:1 normal, 3:1 large + UI).
- [ ] Light mode + dark mode both pass.
- [ ] COI badge background remains identifiable to red-green color-blind users (test with [Color Oracle](https://colororacle.org/)).

## Reduced motion

- [ ] `prefers-reduced-motion: reduce` disables any non-essential animation. Verify in macOS System Settings → Accessibility → Display → Reduce motion.

## Zoom

- [ ] 200% browser zoom: no horizontal scrolling on home, /search, /document/[id].
- [ ] 400% zoom: all content readable, may reflow.

## Forms

- [ ] Every input has an associated `<label>` (or `aria-label`).
- [ ] Errors are announced (use `aria-describedby`) and not communicated by color alone.
