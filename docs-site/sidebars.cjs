const groups = require("./pages.json");
module.exports = { docsSidebar: [
  {type:"doc",id:"index",label:"Visão geral"},
  ...groups.map(([label,ids]) => ({
    type:"category",label,collapsed:label!=="Começar",
    items:ids.map(id=>({type:"doc",id}))
  }))
]};
