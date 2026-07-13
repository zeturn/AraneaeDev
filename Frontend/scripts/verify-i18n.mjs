import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const SRC_DIR = path.resolve(__dirname, '../src');

function walk(dir, ext, out = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'dist') continue;
      walk(full, ext, out);
    } else if (entry.name.endsWith(ext)) out.push(full);
  }
  return out;
}

let problems = 0;
for (const file of walk(SRC_DIR, '.vue')) {
  const lines = fs.readFileSync(file, 'utf8').split('\n');
  const open = lines.findIndex((l) => /^\s*<script\b/.test(l));
  if (open === -1) continue;
  const close = lines.findIndex((l, i) => i > open && /^\s*<\/script>\s*$/.test(l));
  if (close === -1) continue;

  const isSetup = /setup/.test(lines[open]);
  const scriptLines = lines.slice(open + 1, close);
  const scriptText = scriptLines.join('\n');
  const usesBareT = /(^|[^.\w$])t\(/.test(scriptText);
  if (!isSetup || !usesBareT) continue;

  const hasConst = scriptLines.some((l) => l.trim() === 'const { t } = useI18n();');
  const hasImport = /from\s+['"]@\/i18n['"]/.test(lines[open]) ||
    scriptLines.some((l) => /from\s+['"]@\/i18n['"]/.test(l));

  // stray const BEFORE <script>?
  const strayBefore = lines.slice(0, open).some((l) => l.trim() === 'const { t } = useI18n();');

  if (!hasConst || !hasImport || strayBefore) {
    problems++;
    console.log(`PROBLEM: ${file}`);
    console.log(`  hasConst=${hasConst} hasImport=${hasImport} strayBefore=${strayBefore}`);
  }
}
console.log(problems === 0 ? 'OK: all composition scripts are correctly scoped' : `Found ${problems} problems`);
