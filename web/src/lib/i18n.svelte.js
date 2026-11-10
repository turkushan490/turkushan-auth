// Translations for the web app.
//
//   t('lang.label')                      → "Language"
//   t('users.joined', { when: '2 days' }) → fills {when}
//   tn('users.count', 3)                 → picks users.count.one / .other (and .few/.many where a
//                                          language has them) and fills {n}
//
// English (en.json) is always loaded and is the fallback for any missing key. Other
// languages are loaded when picked, so the bundle only carries the one in use.

import en from '../locales/en.json';
import { byCode, languages } from '../locales/index.js';

const loaders = import.meta.glob(['../locales/*.json', '!../locales/en.json']);

export const i18n = $state({ lang: 'en', dict: en });

const STORE_KEY = 'ta-lang';

function fill(s, params) {
  if (!params) return s;
  return s.replace(/\{(\w+)\}/g, (m, k) => (k in params ? String(params[k]) : m));
}

export function t(key, params) {
  const s = i18n.dict[key] ?? en[key];
  return s === undefined ? key : fill(s, params);
}

const plurals = {};
export function tn(key, n, params) {
  const rules = (plurals[i18n.lang] ??= new Intl.PluralRules(i18n.lang));
  const form = rules.select(n);
  const s = i18n.dict[`${key}.${form}`] ?? i18n.dict[`${key}.other`] ?? en[`${key}.${n === 1 ? 'one' : 'other'}`];
  return s === undefined ? key : fill(s, { n: n.toLocaleString(i18n.lang), ...params });
}

// The English text for a server error, or our own translation of its code.
export function errorText(data) {
  if (data?.code && (i18n.dict[`err.${data.code}`] || en[`err.${data.code}`])) return t(`err.${data.code}`);
  return data?.error || t('err.generic');
}

async function load(code) {
  if (code === 'en') return en;
  const loader = loaders[`../locales/${code}.json`];
  if (!loader) return en;
  const mod = await loader();
  return mod.default;
}

function applyDocument(code) {
  document.documentElement.lang = code;
  document.documentElement.dir = byCode[code]?.rtl ? 'rtl' : 'ltr';
}

// The languages a visitor may pick: the admin's list, or all of them.
export function offered(appearance) {
  const allowed = appearance?.languages?.length ? appearance.languages : null;
  return allowed ? languages.filter((l) => allowed.includes(l.code)) : languages;
}

function readStored() {
  try {
    return localStorage.getItem(STORE_KEY) || '';
  } catch {
    return '';
  }
}

// Which language to start in: what this person picked (account, then this browser),
// then the browser's own languages, then the admin's default, then English.
export function pickLanguage(session) {
  const ok = new Set(offered(session?.appearance).map((l) => l.code));
  const tryCode = (c) => {
    if (!c) return '';
    if (ok.has(c)) return c;
    const lower = c.toLowerCase();
    if (lower.startsWith('zh')) {
      const z = /hant|tw|hk|mo/.test(lower) ? 'zh-TW' : 'zh-CN';
      return ok.has(z) ? z : '';
    }
    if (lower.startsWith('nb') || lower.startsWith('nn')) return ok.has('no') ? 'no' : '';
    const base = lower.split('-')[0];
    return ok.has(base) ? base : '';
  };
  for (const c of [session?.user?.lang, readStored(), ...(navigator.languages || [navigator.language])]) {
    const hit = tryCode(c);
    if (hit) return hit;
  }
  return tryCode(session?.appearance?.default_lang) || (ok.has('en') ? 'en' : [...ok][0] || 'en');
}

// setLanguage switches the app. With remember, the choice is kept in this browser
// (and saved on the account by the caller, when signed in).
export async function setLanguage(code, { remember = false } = {}) {
  if (!byCode[code]) code = 'en';
  const dict = await load(code);
  i18n.dict = dict;
  i18n.lang = code;
  applyDocument(code);
  if (remember) {
    try {
      localStorage.setItem(STORE_KEY, code);
    } catch {
      // private window or blocked storage: the choice lasts for this visit
    }
  }
}
