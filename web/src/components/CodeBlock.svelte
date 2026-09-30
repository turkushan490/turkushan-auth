<script>
  let { code, title = '' } = $props();
  let copied = $state(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(code);
    } catch {
      // Fallback for browsers that block the clipboard API.
      const ta = document.createElement('textarea');
      ta.value = code;
      document.body.appendChild(ta);
      ta.select();
      document.execCommand('copy');
      ta.remove();
    }
    copied = true;
    setTimeout(() => (copied = false), 2000);
  }
</script>

<div class="overflow-hidden rounded-lg border border-zinc-800 bg-zinc-950">
  <div class="flex items-center justify-between gap-3 border-b border-zinc-800 px-3 py-2">
    <span class="truncate text-xs text-zinc-500">{title}</span>
    <button
      type="button"
      onclick={copy}
      class="flex shrink-0 items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium transition {copied
        ? 'bg-emerald-500/15 text-emerald-300'
        : 'bg-zinc-800 text-zinc-200 hover:bg-zinc-700'}"
    >
      {#if copied}
        <svg viewBox="0 0 24 24" class="size-3.5" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12l5 5L20 7" /></svg>
        Copied
      {:else}
        <svg viewBox="0 0 24 24" class="size-3.5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="9" y="9" width="11" height="11" rx="2" /><path d="M5 15V5a2 2 0 0 1 2-2h10" /></svg>
        Copy
      {/if}
    </button>
  </div>
  <pre class="overflow-x-auto p-4 font-mono text-xs leading-relaxed text-zinc-300"><code>{code}</code></pre>
</div>
