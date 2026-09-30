<script>
  let {
    label,
    value = $bindable(''),
    type = 'text',
    name,
    autocomplete = 'off',
    required = false,
    optional = false,
    hint = '',
    children,
  } = $props();

  let show = $state(false);
  const id = 'f-' + Math.random().toString(36).slice(2, 9);
  const isPassword = $derived(type === 'password');
</script>

<div>
  <label for={id} class="mb-1.5 flex items-baseline justify-between text-sm font-medium text-zinc-300">
    {label}
    {#if optional}<span class="text-xs font-normal text-zinc-500">optional</span>{/if}
  </label>
  <div class="relative">
    <input
      {id}
      {name}
      {autocomplete}
      {required}
      type={isPassword && show ? 'text' : type}
      bind:value
      autocapitalize="off"
      spellcheck="false"
      class="block w-full rounded-lg border border-zinc-700/80 bg-zinc-950/60 px-3 py-2.5 text-sm text-zinc-100 placeholder-zinc-500 outline-none transition focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/30 {isPassword ? 'pr-10' : ''}"
    />
    {#if isPassword}
      <button
        type="button"
        class="absolute inset-y-0 right-0 grid w-10 place-items-center text-zinc-500 transition hover:text-zinc-300"
        onclick={() => (show = !show)}
        aria-label={show ? 'Hide password' : 'Show password'}
      >
        {#if show}
          <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M3 3l18 18M10.6 10.6a2 2 0 0 0 2.8 2.8M9.9 5.1A10 10 0 0 1 12 5c6 0 9.5 7 9.5 7a17 17 0 0 1-3.2 4.2M6.6 6.6A17 17 0 0 0 2.5 12S6 19 12 19a9.7 9.7 0 0 0 5.4-1.6" />
          </svg>
        {:else}
          <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M2.5 12S6 5 12 5s9.5 7 9.5 7-3.5 7-9.5 7-9.5-7-9.5-7z" />
            <circle cx="12" cy="12" r="3" />
          </svg>
        {/if}
      </button>
    {/if}
  </div>
  {#if hint}
    <p class="mt-1.5 text-xs text-zinc-500">{hint}</p>
  {/if}
  {#if children}
    {@render children()}
  {/if}
</div>
