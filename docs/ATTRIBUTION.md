# Attribution

Sambmin is built on the work of many open-source projects. This document lists all runtime and build-time dependencies with their licenses.

## Primary Dependency

| Project | License | Role |
|---------|---------|------|
| [Samba](https://www.samba.org/) | GPLv3 | AD domain controller implementation; `samba-tool` CLI used for write operations |

## Go Dependencies

From `api/go.mod`:

| Module | Version | License |
|--------|---------|---------|
| [go-ldap/ldap](https://github.com/go-ldap/ldap) | v3.4.12 | MIT |
| [gopkg.in/yaml.v3](https://github.com/go-yaml/yaml) | v3.0.1 | MIT / Apache-2.0 |
| [google/uuid](https://github.com/google/uuid) | v1.6.0 | BSD-3-Clause |
| [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) | v0.36.0 | BSD-3-Clause |
| [Azure/go-ntlmssp](https://github.com/Azure/go-ntlmssp) | v0.0.0-20221128 | MIT |
| [go-asn1-ber/asn1-ber](https://github.com/go-asn1-ber/asn1-ber) | v1.5.8 | MIT |

## Frontend Dependencies

From `web/package.json`:

| Package | Version | License |
|---------|---------|---------|
| [React](https://react.dev/) | 19.2.0 | MIT |
| [React DOM](https://react.dev/) | 19.2.0 | MIT |
| [React Router](https://reactrouter.com/) | 7.13.0 | MIT |
| [Ant Design](https://ant.design/) | 5.29.3 | MIT |
| [Ant Design Icons](https://github.com/ant-design/ant-design-icons) | 6.1.0 | MIT |
| [Ant Design Pro Components](https://procomponents.ant.design/) | 2.8.10 | MIT |
| [D3.js](https://d3js.org/) | 7.9.0 | ISC |
| [cmdk](https://cmdk.paco.me/) | 1.1.1 | MIT |

### Frontend Dev Dependencies

| Package | Version | License |
|---------|---------|---------|
| [TypeScript](https://www.typescriptlang.org/) | 5.9.3 | Apache-2.0 |
| [Vite](https://vite.dev/) | 7.3.1 | MIT |
| [ESLint](https://eslint.org/) | 9.39.1 | MIT |
| [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react) | 5.1.1 | MIT |

## Runtime Dependencies

| Software | License | Role |
|----------|---------|------|
| [PostgreSQL](https://www.postgresql.org/) | PostgreSQL License (MIT-like) | App data storage (audit, sessions, config) |
| [nginx](https://nginx.org/) | BSD-2-Clause | Reverse proxy, static file serving, TLS termination |
| [Python](https://www.python.org/) | PSF License | Script runtime for samba-tool wrappers (stdlib only, no pip packages) |

## Fonts

| Font | License |
|------|---------|
| [Inter](https://rsms.me/inter/) | SIL Open Font License 1.1 |
| [JetBrains Mono](https://www.jetbrains.com/lp/mono/) | SIL Open Font License 1.1 |

## Python Scripts

The Python scripts in `scripts/` use only the Python standard library. No external Python packages are required.

## License Compatibility

All dependencies use licenses compatible with GPLv3:
- **MIT, BSD-2-Clause, BSD-3-Clause, ISC** — Permissive, compatible with GPLv3
- **Apache-2.0** — Compatible with GPLv3 (one-way: Apache code can be included in GPLv3 projects)
- **PostgreSQL License** — MIT-like permissive, compatible with GPLv3
- **SIL OFL 1.1** — Font-specific open license, no conflict with GPLv3
- **PSF License** — Permissive, compatible with GPLv3
