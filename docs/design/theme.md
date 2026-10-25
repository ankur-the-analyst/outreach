# Theme

**Look:** Stitch dashboard (cool gray canvas, terracotta CTAs, 1px borders, tight radii).  
**Type:** Warm Editorial — Noto Serif / IBM Plex Sans / Fira Sans.  
**Not used:** HTML Geist; HTML slate/teal secondary-tertiary; design.md warm-ivory surfaces and `#97472c` primary.

Canonical tokens: `docs/design/theme.css`. Copy into shadcn `globals.css` when the web app is scaffolded. Components use CSS variables only.

| Role | Token | Hex |
|---|---|---|
| Canvas | `--background` | `#F4F4F6` |
| Card / white plane | `--card` | `#FFFFFF` |
| Recessed well | `--surface-container` / `--muted` | `#F0F0F3` |
| Text | `--foreground` | `#1E2024` |
| Meta | `--muted-foreground` | `#6B7280` |
| Border | `--border` / `--outline-variant` | `#E2E2E8` |
| Primary / CTA | `--primary` | `#D9643A` on white text |
| Primary wash | `--primary-fixed` / `--accent` | `#FDEEE9` |
| Secondary | `--secondary` | `#296194` |
| Secondary container | `--secondary-container` | `#92C5FE` |
| Tertiary | `--tertiary` | `#006A61` |
| Tertiary container | `--tertiary-container` | `#36AFA1` |
| Health | `--success` | `#3F7652` |
| Error | `--destructive` | `#BA1A1A` |
| Cards / modules | `--radius-xl` | `0.25rem` |
| Buttons / inputs | `--radius` | `0.25rem` |

Headlines: Noto Serif 600. Body: IBM Plex Sans. Labels: Fira Sans 500. Metrics and timestamps: `tabular-nums`. Draft body max width `68ch`. Lucide icons, stroke ~1.5. Depth via tonal planes + 1px outline; overlay shadow only on menus/modals. Light mode only until a dark Stitch pass exists.
