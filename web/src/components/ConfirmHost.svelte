<script>
  import Modal from './Modal.svelte';
  import { answerConfirm, confirmState } from '../lib/confirm.svelte.js';

  // Closed another way (Esc, clicking outside): that's a "no". After a real
  // answer there is nothing left to answer, so this does nothing.
  $effect(() => {
    if (!confirmState.open) answerConfirm(false);
  });
</script>

<Modal bind:open={confirmState.open} title={confirmState.title}>
  {#if confirmState.text}<p class="text-sm text-zinc-300">{confirmState.text}</p>{/if}
  <div class="mt-6 flex justify-end gap-2">
    <button type="button" data-autofocus
      class="rounded-lg border border-zinc-700 px-3.5 py-2 text-sm font-medium text-zinc-200 transition hover:bg-zinc-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500"
      onclick={() => answerConfirm(false)}>{confirmState.cancel}</button>
    <button type="button"
      class="rounded-lg px-3.5 py-2 text-sm font-semibold transition focus-visible:outline-2 focus-visible:outline-offset-2 {confirmState.danger
        ? 'bg-rose-600 text-white hover:bg-rose-500 focus-visible:outline-rose-500'
        : 'bg-indigo-500 text-[var(--accent-fg,#fff)] hover:bg-indigo-400 focus-visible:outline-indigo-500'}"
      onclick={() => answerConfirm(true)}>{confirmState.ok}</button>
  </div>
</Modal>
