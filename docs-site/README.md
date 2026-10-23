# Nexus documentation portal

Docusaurus 3 is the site engine. All editable content stays in the existing root docs/ directory in the release/v1.0.0 branch. Do not edit generated/.

Local use with Node 22:
1. cd docs-site
2. npm install
3. npm run check:docs
4. npm run start
5. npm run build

The publication allowlist is pages.json. To expose another page, add its documentation ID (relative to docs/, without .md) to the appropriate category. The sidebar and generator share this manifest. Internal validation histories (DEV/, docs/internal/, docs/validation/, docs/superpowers/) are intentionally excluded. Off-menu Markdown links fall back to the source file on GitHub.

The static hosting default is https://iapro-community.github.io/nexus/. The future requested host is https://docs.nexus.iapro.club/, but requires DNS + GitHub Pages configuration. After provisioning, set DOCS_SITE_URL=https://docs.nexus.iapro.club and DOCS_BASE_URL=/ for the build.

Marketing site: https://nexus.iapro.club/. Link it to the documentation paths /quick-start, /product/agents, /product/composer, /product/flow, /product/missions, /product/hud and /release-status after public deployment.

Current documentation describes source code of beta 1.0, not a certified stable release. Canonical status: docs/release-status.md. Source availability does not mean an OSI open-source license (read LICENSE).

CI is currently unavailable; publication workflow is configured but not validated. Before merging, obtain package-lock.json using npm install, commit it, switch the workflow install step to npm ci, run npm run build and inspect mobile/desktop.
