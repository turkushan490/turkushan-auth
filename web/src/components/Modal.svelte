<script>
  let { open = $bindable(false), title, children } = $props();

  function onkeydown(e) {
    if (open && e.key === 'Escape') open = false;
  }
</script>

<svelte:window {onkeydown} />

{#if open}
  <div class="fixed inset-0 z-50 grid place-items-center p-4">
    <button type="button" class="absolute inset-0 cursor-default bg-black/60 backdrop-blur-sm" aria-label="Close" onclick={() => (open = false)}></button>
    <div role="dialog" aria-modal="true" aria-label={title} class="relative w-full max-w-md rounded-2xl border border-zinc-800 bg-zinc-900 p-6 shadow-2xl shadow-black/60">
      <h2 class="text-lg font-semibold tracking-tight">{title}</h2>
      <div class="mt-4">
        {@render children()}
      </div>
    </div>
  </div>
{/if}
