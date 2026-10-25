const site = process.env.DOCS_SITE_URL || "https://iapro-community.github.io";
const base = process.env.DOCS_BASE_URL || "/nexus/";

module.exports = {
  title: "IAPro Nexus",
  tagline: "O Sistema Operacional de Engenharia Multi-Agent",
  url: site,
  baseUrl: base,
  organizationName: "IAPro-Community",
  projectName: "nexus",
  onBrokenLinks: "throw",
  i18n: {
    defaultLocale: "pt-BR",
    locales: ["pt-BR"],
  },
  presets: [
    [
      "classic",
      {
        docs: {
          path: "generated",
          routeBasePath: "/",
          sidebarPath: require.resolve("./sidebars.cjs"),
          editUrl:
            "https://github.com/IAPro-Community/nexus/edit/release/v1.0.0/docs/",
          versions: {
            current: {
              label: "1.0 beta",
              badge: true,
            },
          },
        },
        blog: false,
        theme: {
          customCss: require.resolve("./src/css/custom.css"),
        },
      },
    ],
  ],
  themes: [
    [
      require.resolve("@easyops-cn/docusaurus-search-local"),
      {
        docsDir: "generated",
        // Search indexes the same root route used by the classic docs preset.
        docsRouteBasePath: ["/"],
        hashed: true,
        indexDocs: true,
        indexPages: false,
        indexBlog: false,
      },
    ],
  ],
  themeConfig: {
    navbar: {
      title: "NEXUS / DOCS",
      items: [
        {
          type: "docSidebar",
          sidebarId: "docsSidebar",
          position: "left",
          label: "Documentação",
        },
        {
          to: "/product/conductor",
          label: "Superpoderes",
          position: "left",
        },
        {
          to: "/product/hud",
          label: "HUD & Mascote NEX",
          position: "left",
        },
        {
          to: "/cli",
          label: "CLI",
          position: "left",
        },
        {
          href: "https://nexus.iapro.club/",
          label: "Site oficial",
          position: "right",
        },
        {
          href: "https://github.com/IAPro-Community/nexus",
          label: "GitHub",
          position: "right",
        },
      ],
    },
    footer: {
      style: "dark",
      links: [
        {
          title: "Começar",
          items: [
            { label: "Visão Geral (TL;DR)", to: "/TLDR" },
            { label: "Quick Start", to: "/quick-start" },
            { label: "Instalação Nativa", to: "/getting-started/installation" },
            {
              label: "Docker & Isolamento",
              to: "/getting-started/docker-compat",
            },
          ],
        },
        {
          title: "Superpoderes & Produto",
          items: [
            { label: "Conductor & Autopilot", to: "/product/conductor" },
            { label: "Entrega Autônoma", to: "/product/autonomous-delivery" },
            { label: "HUD Guardian & Mascote NEX", to: "/product/hud" },
            { label: "Composer & Flow", to: "/product/composer" },
          ],
        },
        {
          title: "Referência & Engenharia",
          items: [
            { label: "Manual Completo da CLI", to: "/cli" },
            {
              label: "Matriz Funcional (36 Recursos)",
              to: "/reference/feature-matrix",
            },
            {
              label: "Decisões Arquiteturais (ADRs)",
              to: "/architecture/ADR-agent-interop-projection",
            },
            {
              label: "Padrões de Engenharia",
              to: "/engineering/ENGINEERING_STANDARDS",
            },
          ],
        },
        {
          title: "Projeto & Comunidade",
          items: [
            {
              label: "Código-fonte no GitHub",
              href: "https://github.com/IAPro-Community/nexus",
            },
            { label: "Status da Release", to: "/release-status" },
            { label: "Termos & Licenciamento", to: "/legal/LICENSING" },
            { label: "Segurança", to: "/security.pt-BR" },
          ],
        },
      ],
      copyright:
        "IAPro Community — IAPro Nexus v1.0. Licenciado sob termos oficiais da comunidade.",
    },
    colorMode: {
      defaultMode: "dark",
      respectPrefersColorScheme: true,
    },
  },
};
