# backend/templates/

OVERVIEW
Server-rendered Go html/template files for Project Syrup. Public and admin pages share DaisyUI v5 + Tailwind v4 with a custom "syrup" theme.

STRUCTURE
layouts/
  base.html          — Public layout. Includes PWA meta, offline banner, service worker registration
  admin_base.html    — Admin layout. Dark nav bar, no PWA scripts

partials/
  head.html          — OG tags, CSRF meta token
  header.html        — Public site header with logo
  footer.html        — Shared footer
  admin_nav.html     — Admin nav with mobile hamburger, role-gated "Users" link, logout form

pages/public/
  home.html          — Landing hero
  waffles.html       — Active waffle grid
  waffle_detail.html — Spot grid, claim form, WebSocket client, offline cache
  buyer_stats.html   — Win/loss stats and history for an Instagram handle

pages/admin/
  dashboard.html     — Waffle list with archive/unarchive/delete (type DELETE to confirm)
  waffle_manage.html — Spot grid (bulk pay), pending claims list, winner entry
  waffle_new.html    — Create form with template presets and Instagram media link builder
  reports.html       — Tabbed reports shell (drought, power buyers, monthly, velocity)
  admins.html        — User management table with inline role changes and password reset
  login.html         — Login + forgot password toggle
  reset_password.html — Token-based password reset with visibility toggle

WHERE TO LOOK
Add a public page          → pages/public/*.html, extend base.html
Add an admin page          → pages/admin/*.html, extend admin_base.html
Change nav                 → partials/admin_nav.html or header.html
Modify spot grid styling   → waffle_detail.html + waffle_manage.html (search "spot-grid")
Add report tab             → reports.html + /static/js/reports.js
CSRF token missing         → Check form has `<input type="hidden" name="csrf_token" value="{{ .CSRFToken }}">`

CONVENTIONS
- All pages start with `{{ template "base.html" . }}` or `{{ template "admin_base.html" . }}`
- Blocks: `title`, `content`, `head`, `scripts`
- Partials included via `{{ template "partial_name.html" . }}` (not `{{ partial }}`)
- Form POSTs to admin endpoints must include `csrf_token`
- Use `deref` helper for nullable strings (e.g., `{{ deref .waffle.Description }}`)
- Use `formatDate` helper: `{{ formatDate "Jan 2, 2006" .CreatedAt }}`
- DaisyUI component classes: `card`, `btn`, `badge`, `alert`, `input`, `select`, `tabs`
- Color semantics: success=available, warning=pending, error=paid, secondary=winner, base-300=loser
- Mobile-first: `grid-cols-5 sm:grid-cols-10` for spot grids, `min-h-[44px]` for touch targets
- Inline JS in `{{ define "scripts" }}` blocks; shared JS loaded via `<script src="/static/js/...">`

ANTI-PATTERNS
- Do NOT use `{{ block }}` for partials. Use `{{ template }}` for inclusion.
- Do NOT add inline styles. Use Tailwind utility classes only.
- Do NOT put admin CSRF tokens in public pages. `base.html` does not include CSRF meta.
- Do NOT use `{{ .csrf_token }}` (lowercase). The field is `{{ .CSRFToken }}`.
- Do NOT hardcode theme colors. Use `data-theme="syrup"` and semantic classes.
- Do NOT add new script tags to base.html. Use the `scripts` block in the page template.
- Do NOT forget `disabled` on non-interactive spot buttons. Prevents accidental taps.
