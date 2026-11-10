// confirmDialog: the app's own "Are you sure?" instead of the browser's confirm().
// ConfirmHost (mounted once in App.svelte) shows it.
//
//   if (!(await confirmDialog({ title: 'Block ann?', text: '…', ok: 'Block', danger: true }))) return;
//
// Without ok / cancel the buttons say OK / Cancel in the current language.

import { t } from './i18n.svelte.js';

export const confirmState = $state({ open: false, title: '', text: '', ok: '', cancel: '', danger: false });

let resolver = null;

export function confirmDialog({ title, text = '', ok, cancel, danger = false }) {
  resolver?.(false); // a dialog that was still open counts as cancelled
  Object.assign(confirmState, { open: true, title, text, ok: ok ?? t('common.ok'), cancel: cancel ?? t('common.cancel'), danger });
  return new Promise((resolve) => (resolver = resolve));
}

export function answerConfirm(yes) {
  confirmState.open = false;
  const r = resolver;
  resolver = null;
  r?.(yes);
}
