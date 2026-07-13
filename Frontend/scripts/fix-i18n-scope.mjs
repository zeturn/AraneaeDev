/*
 * fix-i18n-scope.mjs
 *
 * The i18nify transform inserted `import { useI18n } from '@/i18n';` and
 * `const { t } = useI18n();` at the first non-import line of the *whole file*,
 * which (for SFCs whose <script> comes after <template>) lands outside the
 * <script> block. That means `t` is undefined inside the script at runtime
 * (the build still passes because Vite never executes components).
 *
 * This script moves those two lines *inside* the <script> block for every
 * <script setup> (.vue) file that actually uses `t(`. It is idempotent.
 */
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const SRC_DIR = path.resolve(__dirname, '../src');

const IMPORT_LINE = "import { useI18n } from '@/i18n';";
const CONST_LINE = 'const { t } = useI18n();';

function walk(dir, ext, out = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'dist') continue;
      walk(full, ext, out);
    } else if (entry.name.endsWith(ext)) {
      out.push(full);
    }
  }
  return out;
}

let fixed = 0;
for (const file of walk(SRC_DIR, '.vue')) {
  const source = fs.readFileSync(file, 'utf8');
  const lines = source.split('\n');

  const scriptOpenIdx = lines.findIndex((l) => /^\s*<script\b/.test(l));
  if (scriptOpenIdx === -1) continue;
  const scriptCloseIdx = lines.findIndex(
    (l, i) => i > scriptOpenIdx && /^\s*<\/script>\s*$/.test(l)
  );
  if (scriptCloseIdx === -1) continue;

  const scriptOpenLine = lines[scriptOpenIdx];
  const isSetup = /setup/.test(scriptOpenLine);

  // Drop any stray import/const lines that live OUTSIDE the <script> block.
  let cleaned = [];
  for (let i = 0; i < lines.length; i++) {
    if (i < scriptOpenIdx || i > scriptCloseIdx) {
      const t = lines[i].trim();
      if (t === IMPORT_LINE || t === CONST_LINE) continue;
    }
    cleaned.push(lines[i]);
  }

  // Recompute bounds on the cleaned array.
  const openIdx = cleaned.findIndex((l) => /^\s*<script\b/.test(l));
  const closeIdx = cleaned.findIndex(
    (l, i) => i > openIdx && /^\s*<\/script>\s*$/.test(l)
  );

  const scriptContent = cleaned.slice(openIdx + 1, closeIdx).join('\n');
  const usesBareT = /(^|[^.\w$])t\(/.test(scriptContent);
  const needsConst = isSetup && usesBareT;
  if (!needsConst) {
    if (cleaned.length !== lines.length) {
      fs.writeFileSync(file, cleaned.join('\n'), 'utf8');
      fixed++;
    }
    continue;
  }

  const hasImportInside =
    /from\s+['"]@\/i18n['"]/.test(cleaned[openIdx]) ||
    cleaned.slice(openIdx + 1, closeIdx).some((l) => /from\s+['"]@\/i18n['"]/.test(l));
  const hasConstInside = cleaned
    .slice(openIdx + 1, closeIdx)
    .some((l) => l.trim() === CONST_LINE);

  let changed = false;

  // Ensure the import exists inside the <script> block.
  if (!hasImportInside) {
    cleaned.splice(openIdx + 1, 0, IMPORT_LINE);
    changed = true;
  }

  // Recompute close after possible import insertion.
  const closeIdx2 = cleaned.findIndex(
    (l, i) => i > openIdx && /^\s*<\/script>\s*$/.test(l)
  );

  // Ensure `const { t } = useI18n();` exists inside the <script> block,
  // placed right after the import section.
  if (!hasConstInside) {
    let insertAt = -1;
    for (let k = openIdx + 1; k < closeIdx2; k++) {
      if (!/^\s*import\b/.test(cleaned[k])) {
        insertAt = k;
        break;
      }
    }
    if (insertAt === -1) insertAt = closeIdx2; // all imports -> insert before </script>
    cleaned.splice(insertAt, 0, CONST_LINE);
    changed = true;
  }

  if (changed || cleaned.length !== lines.length) {
    fs.writeFileSync(file, cleaned.join('\n'), 'utf8');
    fixed++;
  }
}
console.log(`fixed ${fixed} files`);
