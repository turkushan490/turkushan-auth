<script>
  import { app } from '../lib/state.svelte.js';
</script>

{#if app.toast}
  {#key app.toast.id}
    <div class="pointer-events-none fixed inset-x-0 bottom-4 z-[60] flex justify-center px-4 sm:justify-end">
      <div
        role="status"
        class="pointer-events-auto flex max-w-sm items-center gap-2.5 rounded-xl border px-4 py-3 text-sm shadow-2xl shadow-black/50 backdrop-blur {app.toast.kind === 'error'
          ? 'border-rose-500/30 bg-rose-950/90 text-rose-100'
          : 'border-emerald-500/30 bg-zinc-900/95 text-zinc-100'}"
      >
        {#if app.toast.kind === 'error'}
          <svg viewBox="0 0 24 24" class="size-4 shrink-0 text-rose-400" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><circle cx="12" cy="12" r="9" /><path d="M12 8v5M12 16.5v.5" /></svg>
        {:else}
          <svg viewBox="0 0 24 24" class="size-4 shrink-0 text-emerald-400" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12l5 5L20 7" /></svg>
        {/if}
        {app.toast.text}
      </div>
    </div>
  {/key}
{/if}
