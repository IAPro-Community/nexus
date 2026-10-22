const fs=require("node:fs");
const path=require("node:path");
const groups=require("../pages.json");
const root=path.resolve(__dirname,"../..");
const source=path.join(root,"docs");
const output=path.join(root,"docs-site/generated");
const staticOutput=path.join(root,"docs-site/static");
const files=groups.flatMap((g)=>g[1].map((id)=>id+".md"));
const allowed=new Set(["README.md",...files]);
const check=process.argv.includes("--check");
const github="https://github.com/IAPro-Community/nexus/blob/release/v1.0.0/";
function inside(dir,rel) {
  const full=path.resolve(dir,rel);
  if (!full.startsWith(dir+path.sep)) throw Error("Unsafe path: "+rel);
  return full;
}
function rewrite(text,original) {
  const parent=path.posix.dirname(original);
  return text.replace(/(\]\()([^\s)]+)(\))/g, (all,a,url,b)=>{
    if (/^(https?:|mailto:|tel:|#|data:|\/\/)/i.test(url)) return all;
    const parts=/^([^?#]+)([?#].*)?$/.exec(url);
    if (!parts || parts[1].startsWith("/")) return all;
    const resolved=path.posix.normalize(path.posix.join(parent,parts[1]));
    const suffix=parts[2]||"";
    if (resolved===".."||resolved.startsWith("../")) {
      const outside=path.posix.normalize(path.posix.join("docs",resolved));
      if (outside.startsWith("../")) throw Error("Link outside repo: "+original+" -> "+url);
      return a+github+outside+suffix+b;
    }
    const doc=resolved.startsWith("docs/")?resolved.slice(5):resolved;
    if (doc==="architecture") return a+path.posix.relative(parent,"architecture/overview.md")+suffix+b;
    const target=doc==="README.md"?"index.md":doc;
    if (allowed.has(doc)||doc==="README.md"||doc.startsWith("assets/")) {
      const relative=path.posix.relative(parent,target)||path.posix.basename(target);
      return a+relative+suffix+b;
    }
    if (doc.endsWith(".md")) return a+github+"docs/"+doc+suffix+b;
    return all;
  });
}
function image(page,url) {
  if (/^(https?:|data:|#|\/)/i.test(url)) return;
  const plain=decodeURIComponent(url.split(/[?#]/)[0]);
  const rel=path.posix.normalize(path.posix.join(path.posix.dirname(page),plain));
  if(!rel.startsWith("assets/"))return;
  const src=inside(source,rel);
  if (!fs.statSync(src,{throwIfNoEntry:false})?.isFile())throw Error("Missing image: "+page+" "+rel);
  if (!check) {
    const dst=inside(output,rel);
    fs.mkdirSync(path.dirname(dst),{recursive:true});
    fs.copyFileSync(src,dst);
  }
}
function staticLink(page,url) {
  if (/^(https?:|data:|#|\/)/i.test(url)) return;
  const plain=decodeURIComponent(url.split(/[?#]/)[0]);
  const rel=path.posix.normalize(path.posix.join(path.posix.dirname(page),plain));
  // These files are served from site root, not webpack-managed Markdown assets.
  if (!(/^(?:assets\/.*\.(?:webm|mp4|json)|reference\/features\.json)$/i.test(rel))) return;
  const src=inside(source,rel);
  if (!fs.statSync(src,{throwIfNoEntry:false})?.isFile()) throw Error("Missing static asset: "+page+" "+rel);
  if (!check) {
    const dst=inside(staticOutput,rel);
    fs.mkdirSync(path.dirname(dst),{recursive:true});
    fs.copyFileSync(src,dst);
  }
}
if (allowed.size!==files.length+1)throw Error("Duplicate docs page");
for (const name of allowed) {
  if (!fs.statSync(inside(source,name),{throwIfNoEntry:false})?.isFile()) throw Error("Missing docs/"+name);
}
if (!check) {
  fs.rmSync(output,{recursive:true,force:true});
  fs.mkdirSync(output,{recursive:true});
  fs.rmSync(path.join(staticOutput,"assets"),{recursive:true,force:true});
  fs.rmSync(path.join(staticOutput,"reference"),{recursive:true,force:true});
}
for (const name of allowed) {
  let content=rewrite(fs.readFileSync(inside(source,name),"utf8"),name);
  for (const match of content.matchAll(/!\[[^\]]*\]\(([^)\s]+)(?:\s+[^)]*)?\)/g)) image(name,match[1]);
  for (const match of content.matchAll(/\]\(([^)\s]+)(?:\s+[^)]*)?\)/g)) staticLink(name,match[1]);
  for (const match of content.matchAll(/<(?:img|video|source)\b[^>]*\bsrc=["']([^"']+)["']/g)) image(name,match[1]);
  if (name==="README.md")content="---\nid: index\nslug: /\ntitle: Documentação técnica\n---\n\n"+content;
  if (!check) {
    const to=inside(output,name==="README.md"?"index.md":name);
    fs.mkdirSync(path.dirname(to),{recursive:true});
    fs.writeFileSync(to,content,"utf8");
  }
}
console.log("Nexus Docs: "+allowed.size+" selected Markdown pages checked.");
