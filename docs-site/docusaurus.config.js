const site = process.env.DOCS_SITE_URL || "https://iapro-community.github.io";
const base = process.env.DOCS_BASE_URL || "/nexus/";
module.exports = {
  title:"Nexus",tagline:"Documentação técnica",url:site,baseUrl:base,
  organizationName:"IAPro-Community",projectName:"nexus",
  onBrokenLinks:"throw",
  i18n:{defaultLocale:"pt-BR",locales:["pt-BR"]},
  presets:[["classic",{
    docs:{
      path:"generated",
      routeBasePath:"/",
      sidebarPath:require.resolve("./sidebars.cjs"),
      editUrl:"https://github.com/IAPro-Community/nexus/edit/release/v1.0.0/docs/",
      versions:{current:{label:"1.0 beta",badge:true}}
    },
    blog:false,
    theme:{customCss:require.resolve("./src/css/custom.css")}
  }]],
  themes:[[require.resolve("@easyops-cn/docusaurus-search-local"),
    {hashed:true,indexDocs:true,indexPages:false,indexBlog:false}]],
  themeConfig:{
    navbar:{title:"NEXUS / DOCS",items:[
      {type:"docSidebar",sidebarId:"docsSidebar",position:"left",label:"Documentação"},
      {href:"https://nexus.iapro.club/",label:"Site oficial",position:"right"},
      {href:"https://github.com/IAPro-Community/nexus",label:"GitHub",position:"right"}
    ]},
    footer:{style:"dark",links:[
      {title:"Nexus",items:[{label:"Quick Start",to:"/quick-start"},
        {label:"Status da release",to:"/release-status"}]},
      {title:"Comunidade",items:[{label:"Código-fonte",href:"https://github.com/IAPro-Community/nexus"},
        {label:"Licença",to:"/legal/LICENSING"}]}
    ],copyright:"IAPro Community — consulte as condições de licenciamento."},
    colorMode:{defaultMode:"dark",respectPrefersColorScheme:true}
  }
};
