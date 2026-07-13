/*
 * i18nify.mjs — mechanically wraps user-facing Chinese strings in Araneae .vue files.
 *
 * Strategy
 *  - Template text nodes / attributes containing CJK  -> $t('...')   (global, works in both APIs)
 *  - <script setup> CJK string literals               -> t('...')     (needs useI18n)
 *  - Options API (<script> without setup) CJK literals-> this.$t('...') (global $t)
 *
 * Keys are the original Chinese strings (source language). zh-CN is an identity map;
 * en-US translations are provided separately by the developer.
 *
 * Usage:
 *   node scripts/i18nify.mjs            # transform in place + write catalogs
 *   node scripts/i18nify.mjs --dry      # collect keys only, no file writes
 */

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const SRC_DIR = path.resolve(__dirname, '../src');
const MESSAGES_DIR = path.resolve(SRC_DIR, 'i18n/messages');
const DRY = process.argv.includes('--dry');

const CJK = /[㐀-鿿　-〿＀-￯]/;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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

function escapeForTemplate(key) {
  // keys are CJK text; they won't contain quotes in practice.
  return key;
}

// Evaluate a JS string literal's inner content to its real value so the
// catalog key matches what `t(...)` receives at runtime.
function evalString(inner, q) {
  if (q === '"') {
    try {
      return JSON.parse('"' + inner + '"');
    } catch (e) {
      return inner;
    }
  }
  try {
    const normalized = inner
      .replace(/\\'/g, '\\"')
      .replace(/"/g, '\\"');
    return JSON.parse('"' + normalized + '"');
  } catch (e) {
    return inner;
  }
}

function isInsideComment(code, index) {
  const before = code.slice(0, index);
  const lineStart = before.lastIndexOf('\n') + 1;
  if (before.slice(lineStart).includes('//')) return true;
  const opens = (before.match(/\/\*/g) || []).length;
  const closes = (before.match(/\*\//g) || []).length;
  if (opens > closes) return true;
  return false;
}

// Wrap CJK string literals in `code` using `wrap(inner, quote)`.
// Skips object keys (followed by unescaped ':') and comments.
function replaceCJKLiterals(code, wrap) {
  const quoteRe = /(['"])((?:\\.|(?!\1).)*)\1/gs;
  const out = [];
  let last = 0;
  let m;
  while ((m = quoteRe.exec(code)) !== null) {
    const full = m[0];
    const q = m[1];
    const inner = m[2];
    const start = m.index;
    const end = start + full.length;
    out.push(code.slice(last, start));
    if (CJK.test(inner)) {
      let k = end;
      while (k < code.length && /\s/.test(code[k])) k++;
      if (code[k] === ':' && code[k + 1] !== ':') {
        out.push(full); // object key -> leave untouched
      } else if (isInsideComment(code, start)) {
        out.push(full); // comment -> leave untouched
      } else {
        out.push(wrap(inner, q));
      }
    } else {
      out.push(full);
    }
    last = end;
  }
  out.push(code.slice(last));
  return out.join('');
}

// Find the { ... } block of `export default { ... }` in Options API scripts.
function findExportDefaultBlock(code) {
  const idx = code.indexOf('export default');
  if (idx === -1) return null;
  let i = code.indexOf('{', idx);
  if (i === -1) return null;
  let depth = 0;
  let j = i;
  for (; j < code.length; j++) {
    const ch = code[j];
    if (ch === '{') depth++;
    else if (ch === '}') {
      depth--;
      if (depth === 0) break;
    }
  }
  return { start: i, end: j + 1, body: code.slice(i, j + 1) };
}

// ---------------------------------------------------------------------------
// Template transform
// ---------------------------------------------------------------------------

const TRANSLATE_ATTRS = [
  'placeholder',
  'label',
  'title',
  'content',
  'aria-label',
  'confirm-button-text',
  'cancel-button-text',
  'ok-text',
  'cancel-text',
];

function transformTemplate(body, keys) {
  let changed = false;

  // 1) text nodes between tags that contain CJK
  body = body.replace(/(>)([^<]+?)(<)/g, (full, open, text, close) => {
    if (/[<>{}&]/.test(text)) return full; // unsafe / dynamic -> skip
    if (!CJK.test(text)) return full;
    const m = text.match(/^(\s*)([\s\S]*?)(\s*)$/);
    const lead = m[1];
    const inner = m[2];
    const trail = m[3];
    if (!inner.trim()) return full;
    keys.add(inner);
    changed = true;
    return `${open}${lead}{{ $t('${escapeForTemplate(inner)}') }}${trail}${close}`;
  });

  // 2) attributes with CJK values
  const attrRe = new RegExp(
    '(\\s)(' + TRANSLATE_ATTRS.join('|') + ')=("([^"]*)"|\'([^\']*)\')',
    'g'
  );
  body = body.replace(attrRe, (full, sp, attr, val, dq, sq) => {
    const raw = dq !== undefined ? dq : sq;
    if (!CJK.test(raw)) return full;
    keys.add(raw);
    changed = true;
    return `${sp}:${attr}="$t('${escapeForTemplate(raw)}')"`;
  });

  return { body, changed };
}

// ---------------------------------------------------------------------------
// Script transform
// ---------------------------------------------------------------------------

function transformScript(attrs, body, keys) {
  const isSetup = /setup/.test(attrs);
  let changed = false;

  if (isSetup) {
    const wrapped = replaceCJKLiterals(body, (inner, q) => {
      keys.add(evalString(inner, q));
      changed = true;
      return `t(${q}${inner}${q})`;
    });
    if (!changed) return { body, changed };
    let result = wrapped;
    // ensure useI18n is imported and `t` is in scope
    if (!/from\s+['"]@\/i18n['"]/.test(result)) {
      const lines = result.split('\n');
      let firstNonImport = lines.findIndex(
        (l) => !/^\s*import\s/.test(l) && l.trim() !== ''
      );
      if (firstNonImport === -1) firstNonImport = lines.length;
      lines.splice(firstNonImport, 0, 'const { t } = useI18n();');
      lines.unshift("import { useI18n } from '@/i18n';");
      result = lines.join('\n');
    } else if (!/const\s*\{\s*t\s*\}\s*=\s*useI18n/.test(result)) {
      const lines = result.split('\n');
      let firstNonImport = lines.findIndex(
        (l) => !/^\s*import\s/.test(l) && l.trim() !== ''
      );
      if (firstNonImport === -1) firstNonImport = lines.length;
      lines.splice(firstNonImport, 0, 'const { t } = useI18n();');
      result = lines.join('\n');
    }
    return { body: result, changed };
  }

  // Options API: wrap CJK literals inside export default { ... } with this.$t
  const block = findExportDefaultBlock(body);
  if (!block) {
    // no export default; still try whole-body (rare for options api)
    const wrapped = replaceCJKLiterals(body, (inner, q) => {
      keys.add(evalString(inner, q));
      changed = true;
      return `this.$t(${q}${inner}${q})`;
    });
    return { body: wrapped, changed };
  }
  const innerWrapped = replaceCJKLiterals(block.body, (inner, q) => {
    keys.add(evalString(inner, q));
    changed = true;
    return `this.$t(${q}${inner}${q})`;
  });
  const newBody = body.slice(0, block.start) + innerWrapped + body.slice(block.end);
  return { body: newBody, changed };
}

// ---------------------------------------------------------------------------
// File processing
// ---------------------------------------------------------------------------

function transformFile(file, keys) {
  const source = fs.readFileSync(file, 'utf8');
  let result = source;
  let fileChanged = false;

  // template block
  result = result.replace(/<template>([\s\S]*?)<\/template>/, (m, body) => {
    const { body: nb, changed } = transformTemplate(body, keys);
    if (changed) fileChanged = true;
    return `<template>${nb}</template>`;
  });

  // script blocks (could be multiple, e.g. setup + normal — rare)
  result = result.replace(
    /<script\b([^>]*)>([\s\S]*?)<\/script>/g,
    (m, attrs, body) => {
      const { body: nb, changed } = transformScript(attrs, body, keys);
      if (changed) fileChanged = true;
      return `<script${attrs}>${nb}</script>`;
    }
  );

  if (!DRY && fileChanged) {
    fs.writeFileSync(file, result, 'utf8');
  }
  return fileChanged;
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

function main() {
  const files = walk(SRC_DIR, '.vue');
  const keys = new Set();
  let changedFiles = 0;

  for (const file of files) {
    // never touch our own message catalog files (they are .js anyway)
    if (transformFile(file, keys)) changedFiles++;
  }

  const keyList = Array.from(keys).sort();

  if (DRY) {
    fs.writeFileSync(
      path.resolve(__dirname, 'i18n-keys.json'),
      JSON.stringify(keyList, null, 2),
      'utf8'
    );
    console.log(`[dry] scanned ${files.length} .vue files`);
    console.log(`[dry] collected ${keyList.length} unique CJK keys`);
    console.log(`[dry] keys written to scripts/i18n-keys.json`);
    return;
  }

  // write zh-CN.js (identity map) and en-US.js stub
  if (!fs.existsSync(MESSAGES_DIR)) fs.mkdirSync(MESSAGES_DIR, { recursive: true });
  const zhEntries = keyList.map((k) => `  ${JSON.stringify(k)}: ${JSON.stringify(k)},`).join('\n');
  const enEntries = keyList
    .map((k) => `  ${JSON.stringify(k)}: ${JSON.stringify(k)}, // TODO: translate`)
    .join('\n');
  fs.writeFileSync(
    path.resolve(MESSAGES_DIR, 'zh-CN.js'),
    `// i18n message catalog: Simplified Chinese (source language, identity map).\n// Keys are the original Chinese strings used across the UI.\nexport default {\n${zhEntries}\n};\n`,
    'utf8'
  );
  fs.writeFileSync(
    path.resolve(MESSAGES_DIR, 'en-US.js'),
    `// i18n message catalog: English (United States).\n// Keys are the original Chinese strings; values are their English translations.\nexport default {\n${enEntries}\n};\n`,
    'utf8'
  );

  console.log(`transformed ${changedFiles}/${files.length} .vue files`);
  console.log(`wrote ${keyList.length} keys to zh-CN.js / en-US.js`);
}

main();
