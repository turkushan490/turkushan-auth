<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import { app, link } from '../lib/state.svelte.js';

  const user = $derived(app.session.user);
</script>

<AuthLayout title="You're signed in">
  <div class="space-y-5">
    <div class="flex items-center gap-3 rounded-xl border border-zinc-800 bg-zinc-950/50 p-4">
      <div class="grid size-11 shrink-0 place-items-center rounded-full bg-indigo-500/20 text-base font-semibold uppercase text-indigo-300">
        {user.username.slice(0, 1)}
      </div>
      <div class="min-w-0">
        <div class="flex items-center gap-2">
          <p class="truncate font-medium">{user.username}</p>
          {#if user.is_admin}
            <span class="rounded-full bg-indigo-500/15 px-2 py-0.5 text-xs font-medium text-indigo-300 ring-1 ring-indigo-400/20">Admin</span>
          {/if}
        </div>
        <p class="truncate text-sm text-zinc-400">
          {#if user.email}
            {user.email}
            {#if !user.email_verified}<span class="text-amber-400/90">· not verified</span>{/if}
          {:else}
            No email address
          {/if}
        </p>
      </div>
    </div>

    <p class="text-sm text-zinc-400">
      You can now open the sites you have access to. This login works on all of them.
    </p>

    <div class="space-y-2">
      {#if user.is_admin}
        <a
          href="/admin"
          onclick={link}
          class="flex w-full items-center justify-center gap-2 rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white shadow-lg shadow-indigo-500/20 transition hover:bg-indigo-400"
        >
          <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M12 3l8 3v6c0 4.5-3.2 8.3-8 9-4.8-.7-8-4.5-8-9V6z" />
          </svg>
          Admin panel
        </a>
      {/if}
      <a
        href="/logout"
        onclick={link}
        class="flex w-full items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm font-medium text-zinc-100 transition hover:bg-zinc-700"
      >
        <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9" />
        </svg>
        Sign out
      </a>
    </div>
  </div>
</AuthLayout>
