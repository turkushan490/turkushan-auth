import { t } from './i18n.svelte.js';

// Same rules as the server (internal/auth/validate.go).
export function passwordRules(pw) {
  return [
    { ok: [...pw].length >= 6, text: t('pw.length') },
    { ok: /\p{Lu}/u.test(pw), text: t('pw.capital') },
    { ok: /[^\p{L}\p{N}\s]/u.test(pw), text: t('pw.symbol') },
  ];
}
