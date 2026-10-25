import { createHash } from 'node:crypto';
import postcss from 'postcss';
import selectorParser from 'postcss-selector-parser';

export function scopeCssModule(compiledCss, moduleIdentity) {
  const fileHash = createHash('md5').update(moduleIdentity).digest('hex').slice(0, 6);
  const classMap = Object.create(null);
  const stylesheet = postcss.parse(compiledCss);
  stylesheet.walkRules((rule) => {
    rule.selector = selectorParser((selectors) => {
      selectors.walkClasses((classNode) => {
        const className = classNode.value;
        const scoped = `${className}_${fileHash}`;
        classMap[className] = scoped;
        classNode.value = scoped;
      });
    }).processSync(rule.selector);
  });

  return { css: stylesheet.toString(), classMap };
}
