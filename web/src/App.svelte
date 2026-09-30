<script>
  import { app, refreshSession } from './lib/state.svelte.js';
  import Login from './pages/Login.svelte';
  import Register from './pages/Register.svelte';
  import Account from './pages/Account.svelte';
  import Logout from './pages/Logout.svelte';
  import Pending from './pages/Pending.svelte';
  import Admin from './pages/admin/Admin.svelte';
  import Toast from './components/Toast.svelte';

  refreshSession();

  const page = $derived.by(() => {
    const signedIn = app.session?.authenticated;
    const path = app.path;
    if (path === '/login') return 'login';
    if (path === '/register') return 'register';
    if (path === '/logout') return 'logout';
    if (!signedIn) return 'login';
    if (path === '/pending') return 'pending';
    if (path === '/admin' || path.startsWith('/admin/')) return 'admin';
    return 'account';
  });
</script>

{#if app.loading}
  <div class="grid min-h-screen place-items-center bg-zinc-950">
    <svg class="size-6 animate-spin text-zinc-600" viewBox="0 0 24 24" fill="none" aria-label="Loading">
      <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" class="opacity-25" />
      <path d="M22 12a10 10 0 0 0-10-10" stroke="currentColor" stroke-width="3" stroke-linecap="round" />
    </svg>
  </div>
{:else if page === 'login'}
  {#key app.path}<Login />{/key}
{:else if page === 'register'}
  <Register />
{:else if page === 'logout'}
  <Logout />
{:else if page === 'pending'}
  <Pending />
{:else if page === 'admin'}
  <Admin />
{:else}
  <Account />
{/if}

<Toast />
