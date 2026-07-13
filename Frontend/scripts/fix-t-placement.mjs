/*
 * fix-t-placement.mjs
 *
 * The i18nify transform inserted `const { t } = useI18n();` right after the
 * first line that does not *start* with the word `import`. For files with a
 * multi-line import statement that left the line inside the import block, which
 * is a syntax error. This script removes the (possibly misplaced) inserted
 * lines and re-inserts them correctly, after the full import section.
 */
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
    } else if (entry.name.endsWith(ext)) {
      out.push(full);
    }
  }
  return out;
}

function findFirstNonImportLine(lines) {
  let inImport = false;
  for (let i = 0; i < lines.length; i++) {
    const l = lines[i];
    if (!inImport && /^\s*import\b/.test(l)) {
      inImport = true;
    }
    if (inImport) {
      if (/;/.test(l) || /from\s*['"]/.test(l)) inImport = false;
      continue;
    }
    if (l.trim() === '') continue;
    return i;
  }
  return lines.length;
}

const IMPORT_LINE = "import { useI18n } from '@/i18n';";
const CONST_LINE = 'const { t } = useI18n();';

let fixed = 0;
for (const file of walk(SRC_DIR, '.vue')) {
  const source = fs.readFileSync(file, 'utf8');
  const lines = source.split('\n');
  const hasImport = lines.some((l) => l.trim() === IMPORT_LINE);
  const hasConst = lines.some((l) => l.trim() === CONST_LINE);
  if (!hasImport && !hasConst) continue;

  const filtered = lines.filter(
    (l) => l.trim() !== IMPORT_LINE && l.trim() !== CONST_LINE
  );

  const idx = findFirstNonImportLine(filtered);
  filtered.splice(idx, 0, CONST_LINE);

  const alreadyImports = filtered.some((l) => /from\s+['"]@\/i18n['"]/.test(l));
  if (!alreadyImports) filtered.unshift(IMPORT_LINE);

  fs.writeFileSync(file, filtered.join('\n'), 'utf8');
  fixed++;
}
console.log(`fixed ${fixed} files`);
