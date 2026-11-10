import { i18n, t } from './i18n.svelte.js';

export function timeAgo(unix) {
  if (!unix) return t('time.never');
  const rtf = new Intl.RelativeTimeFormat(i18n.lang, { numeric: 'auto' });
  const s = Math.floor(Date.now() / 1000) - unix;
  if (s < 60) return rtf.format(0, 'second');
  const m = Math.floor(s / 60);
  if (m < 60) return rtf.format(-m, 'minute');
  const h = Math.floor(m / 60);
  if (h < 24) return rtf.format(-h, 'hour');
  const d = Math.floor(h / 24);
  if (d < 30) return rtf.format(-d, 'day');
  return new Date(unix * 1000).toLocaleDateString(i18n.lang);
}

export function fullDate(unix) {
  return unix ? new Date(unix * 1000).toLocaleString(i18n.lang) : '';
}
