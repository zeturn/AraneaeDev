// Writes src/i18n/messages/en-US.js by pairing the keys from zh-CN.js with the
// English translations provided in translations.mjs. Keys without a translation
// fall back to the Chinese source string so the app never breaks.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import zhCN from '../src/i18n/messages/zh-CN.js';
import { translations } from './translations.mjs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const outPath = path.resolve(__dirname, '../src/i18n/messages/en-US.js');

const entries = Object.keys(zhCN).map((key) => {
  const en = translations[key] !== undefined ? translations[key] : key;
  return `  ${JSON.stringify(key)}: ${JSON.stringify(en)},`;
});

const content =
  '// i18n message catalog: English (United States).\n' +
  '// Keys are the original Chinese strings; values are their English translations.\n' +
  'export default {\n' +
  entries.join('\n') +
  '\n};\n';

fs.writeFileSync(outPath, content, 'utf8');

const missing = Object.keys(zhCN).filter((k) => translations[k] === undefined).length;
console.log(`wrote en-US.js with ${Object.keys(zhCN).length} keys (${missing} fell back to Chinese)`);
