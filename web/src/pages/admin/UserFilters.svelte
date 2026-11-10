<script>
  import { card } from './ui.js';

  let {
    data,
    search = $bindable(''),
    siteFilter = $bindable(''),
    groupFilter = $bindable(''), // group id, 'admins' or 'none'
    loginFilter = $bindable(''), // '' | password | discord | google
    statusFilter = $bindable(''),
    sortBy = $bindable('name'),
  } = $props();

  const statusFilters = [
    { id: '', name: 'Any status' },
    { id: 'pending', name: 'Has open request' },
    { id: 'blocked', name: 'Blocked' },
    { id: 'locked', name: 'Locked' },
    { id: 'noemail', name: 'No email' },
    { id: 'unverified', name: 'Email not verified' },
    { id: 'nosites', name: "Can't open any site" },
  ];
  const sorts = [
    { id: 'name', name: 'Name (A–Z)' },
    { id: 'newest', name: 'Newest first' },
    { id: 'oldest', name: 'Oldest first' },
    { id: 'login', name: 'Last login' },
  ];

  const select = 'rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-sm text-zinc-200 outline-none focus:border-indigo-500';
</script>

<div class="{card} mb-4 grid gap-2 p-3 sm:grid-cols-2 lg:grid-cols-5">
  <label class="sr-only" for="user-search">Search users</label>
  <input
    id="user-search"
    type="search"
    placeholder="Search name, email or group…"
    bind:value={search}
    class="rounded-lg border border-zinc-800 bg-zinc-950/60 px-3 py-1.5 text-sm outline-none placeholder:text-zinc-500 focus:border-indigo-500 sm:col-span-2 lg:col-span-5"
  />
  <select bind:value={siteFilter} class={select} aria-label="Filter by site">
    <option value="">All sites</option>
    {#each data.sites as s (s.id)}<option value={String(s.id)}>{s.name}</option>{/each}
  </select>
  <select bind:value={groupFilter} class={select} aria-label="Filter by group">
    <option value="">All groups</option>
    <option value="admins">Admins</option>
    {#each data.groups as g (g.id)}<option value={String(g.id)}>{g.name}</option>{/each}
    <option value="none">Not in any group</option>
  </select>
  <select bind:value={loginFilter} class={select} aria-label="Filter by login type">
    <option value="">Any login</option>
    <option value="password">Username &amp; password</option>
    <option value="discord">Discord</option>
    <option value="google">Google</option>
  </select>
  <select bind:value={statusFilter} class={select} aria-label="Filter by status">
    {#each statusFilters as f}<option value={f.id}>{f.name}</option>{/each}
  </select>
  <select bind:value={sortBy} class={select} aria-label="Sort">
    {#each sorts as s}<option value={s.id}>Sort: {s.name}</option>{/each}
  </select>
</div>
