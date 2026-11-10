// Checks the translations in src/locales against en.json:
//   - every key used in the code with t('…') / tn('…') exists in en.json (error)
//   - each language has the same keys as en.json (missing keys: warning, English is shown)
//   - no unknown keys, and the same {placeholders} as English (error)
//   - every language in index.js has a file, and the other way round (error)
//
//   node scripts/check-locales.js          (exit code 1 on errors)
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..', 'src');
const dir = join(root, 'locales');
const en = JSON.parse(readFileSync(join(dir, 'en.json'), 'utf8'));
const { languages } = await import('file://' + join(dir, 'index.js').replace(/\\/g, '/'));

let errors = 0;
let warnings = 0;
const error = (msg) => (errors++, console.error('error:', msg));
const warn = (msg) => (warnings++, console.warn('warning:', msg));
const placeholders = (s) => [...String(s).matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort().join(',');

// Keys used in the code.
function walk(d) {
  return readdirSync(d).flatMap((f) => {
    const p = join(d, f);
    if (statSync(p).isDirectory()) return f === 'locales' ? [] : walk(p);
    return /\.(svelte|js)$/.test(f) ? [p] : [];
  });
}
const pluralForms = ['zero', 'one', 'two', 'few', 'many', 'other'];
for (const file of walk(root)) {
  const src = readFileSync(file, 'utf8');
  for (const m of src.matchAll(/\bt\(\s*'([a-z0-9_.-]+)'/g)) {
    if (!(m[1] in en)) error(`${file.slice(root.length + 1)} uses missing key '${m[1]}'`);
  }
  for (const m of src.matchAll(/\btn\(\s*'([a-z0-9_.-]+)'/g)) {
    if (!(`${m[1]}.other` in en)) error(`${file.slice(root.length + 1)} uses plural '${m[1]}' without '${m[1]}.other'`);
  }
}

const codes = new Set(languages.map((l) => l.code));
const files = readdirSync(dir).filter((f) => f.endsWith('.json')).map((f) => f.slice(0, -5));
for (const f of files) if (!codes.has(f)) error(`${f}.json is not listed in locales/index.js`);

for (const { code } of languages) {
  if (code === 'en') continue;
  if (!files.includes(code)) {
    error(`no ${code}.json for a language listed in index.js`);
    continue;
  }
  let dict;
  try {
    dict = JSON.parse(readFileSync(join(dir, `${code}.json`), 'utf8'));
  } catch (e) {
    error(`${code}.json is not valid JSON: ${e.message}`);
    continue;
  }
  const missing = Object.keys(en).filter((k) => !(k in dict) && !pluralForms.some((p) => k.endsWith('.' + p) && k !== k.replace(/\.[a-z]+$/, '.other')));
  if (missing.length) warn(`${code}: ${missing.length} missing (shown in English): ${missing.slice(0, 5).join(', ')}${missing.length > 5 ? ', …' : ''}`);
  for (const [k, v] of Object.entries(dict)) {
    const base = k.replace(/\.(zero|two|few|many)$/, '.other');
    const ref = en[k] ?? en[base];
    if (ref === undefined) {
      error(`${code}: unknown key '${k}'`);
      continue;
    }
    // A piece that is only punctuation in English (like ".") may be blank, e.g. in Thai.
    if (typeof v !== 'string' || v === '' || (!v.trim() && /[\p{L}\p{N}]/u.test(ref))) error(`${code}: '${k}' is empty`);
    else if (placeholders(v) !== placeholders(ref) && !(k.endsWith('.one') && placeholders(v) === '')) {
      error(`${code}: '${k}' has {${placeholders(v)}} but English has {${placeholders(ref)}}`);
    }
  }
}

console.log(`${languages.length} languages, ${Object.keys(en).length} keys: ${errors} errors, ${warnings} warnings`);
process.exit(errors ? 1 : 0);
