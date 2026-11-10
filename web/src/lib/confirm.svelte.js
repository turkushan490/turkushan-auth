// confirmDialog: the app's own "Are you sure?" instead of the browser's confirm().
// ConfirmHost (mounted once in App.svelte) shows it.
//
//   if (!(await confirmDialog({ title: 'Block ann?', text: '…', ok: 'Block', danger: true }))) return;

export const confirmState = $state({ open: false, title: '', text: '', ok: 'OK', cancel: 'Cancel', danger: false });

let resolver = null;

export function confirmDialog({ title, text = '', ok = 'OK', cancel = 'Cancel', danger = false }) {
  resolver?.(false); // a dialog that was still open counts as cancelled
  Object.assign(confirmState, { open: true, title, text, ok, cancel, danger });
  return new Promise((resolve) => (resolver = resolve));
}

export function answerConfirm(yes) {
  confirmState.open = false;
  const r = resolver;
  resolver = null;
  r?.(yes);
}
