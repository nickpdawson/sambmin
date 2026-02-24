# Sambmin Frontend

React 19 + TypeScript + Vite + Ant Design 5 frontend for Sambmin.

## Development

```bash
npm install
npm run dev      # Vite dev server on http://localhost:5173
```

The dev server proxies `/api/` requests to the Go backend at `http://localhost:8443`. Start the backend first (mock mode works fine for frontend development):

```bash
cd ../api && go run ./cmd/sambmin/
```

## Production Build

```bash
npm run build    # Output to dist/
```

Deploy `dist/` to your web server's document root. nginx or Apache serves static files and proxies `/api/` to the Go backend.

## Structure

```
src/
├── api/            # Typed API client (fetch wrapper, CSRF handling)
├── components/     # Shared UI components
├── pages/          # Page-level components (Users, Groups, DNS, etc.)
└── App.tsx         # Router, layout shell, auth context
```

## Key Libraries

- **[Ant Design 5](https://ant.design/)** — UI component library
- **[Ant Design Pro Components](https://procomponents.ant.design/)** — ProTable, ProForm for data-heavy views
- **[D3.js](https://d3js.org/)** — Replication topology visualization
- **[cmdk](https://cmdk.paco.me/)** — Command palette (Cmd+K)
- **[React Router 7](https://reactrouter.com/)** — Client-side routing

## Design

- **Body text**: Inter
- **Technical values** (DNs, SIDs, IPs): JetBrains Mono
- **Locale**: `en_US` on Ant Design ConfigProvider (prevents Chinese text in ProTable)

## Linting

```bash
npm run lint
```
