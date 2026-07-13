# DevSignal frontend

The React 19 + TypeScript single-page app for DevSignal, built with Vite.

```bash
npm install     # once
npm run dev     # dev server on http://localhost:5173 (proxies /api to :8080)
npm run build   # type-check (tsc -b) + production bundle in dist/
npm run lint    # oxlint
```

The dev server proxies `/api` to the Go backend on `localhost:8080`
(see `vite.config.ts`), so run the API alongside it. In production the Go
server serves the `dist/` build itself — see the repository root
[Dockerfile](../Dockerfile) and [docs/MAINTENANCE_GUIDE.md](../docs/MAINTENANCE_GUIDE.md).

Source layout: `src/api` (typed client, query hooks, domain types),
`src/auth` (session context), `src/components` (design-system pieces and
charts), `src/pages` (one file per screen), `src/lib` (formatting helpers),
`src/styles/app.css` (design tokens and component classes).
