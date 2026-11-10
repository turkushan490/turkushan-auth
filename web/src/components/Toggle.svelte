<script>
  let { checked = $bindable(false), label, description = '', disabled = false, onchange } = $props();

  function flip() {
    if (disabled) return;
    checked = !checked;
    onchange?.(checked);
  }
</script>

<button
  type="button"
  role="switch"
  aria-checked={checked}
  {disabled}
  onclick={flip}
  class="flex w-full items-start justify-between gap-4 rounded-md text-start focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
>
  <span class="min-w-0">
    <span class="block text-sm font-medium text-zinc-200">{label}</span>
    {#if description}<span class="mt-0.5 block text-xs text-zinc-500">{description}</span>{/if}
  </span>
  <span class="relative mt-0.5 inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors {checked ? 'bg-indigo-500' : 'bg-zinc-700'}">
    <span class="inline-block size-4 rounded-full bg-white shadow transition-transform {checked ? 'translate-x-[18px] rtl:-translate-x-[18px]' : 'translate-x-0.5 rtl:-translate-x-0.5'}"></span>
  </span>
</button>
