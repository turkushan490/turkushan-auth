<script>
  import { card } from './ui.js';
  import { t } from '../../lib/i18n.svelte.js';

  let {
    data,
    search = $bindable(''),
    siteFilter = $bindable(''),
    groupFilter = $bindable(''), // group id, 'admins' or 'none'
    loginFilter = $bindable(''), // '' | password | discord | google
    statusFilter = $bindable(''),
    sortBy = $bindable('name'),
  } = $props();

  const statusFilters = $derived([
    { id: '', name: t('users.filter_any_status') },
    { id: 'pending', name: t('users.filter_pending') },
    { id: 'blocked', name: t('users.filter_blocked') },
    { id: 'locked', name: t('users.filter_locked') },
    { id: 'noemail', name: t('users.filter_noemail') },
    { id: 'unverified', name: t('users.filter_unverified') },
    { id: 'nosites', name: t('users.filter_nosites') },
  ]);
  const sorts = $derived([
    { id: 'name', name: t('users.sort_name') },
    { id: 'newest', name: t('users.sort_newest') },
    { id: 'oldest', name: t('users.sort_oldest') },
    { id: 'login', name: t('users.sort_login') },
  ]);

  const select = 'rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-sm text-zinc-200 outline-none focus:border-indigo-500';
</script>

<div class="{card} mb-4 grid gap-2 p-3 sm:grid-cols-2 lg:grid-cols-5">
  <label class="sr-only" for="user-search">{t('users.search_label')}</label>
  <input
    id="user-search"
    type="search"
    placeholder={t('users.search_placeholder')}
    bind:value={search}
    class="rounded-lg border border-zinc-800 bg-zinc-950/60 px-3 py-1.5 text-sm outline-none placeholder:text-zinc-500 focus:border-indigo-500 sm:col-span-2 lg:col-span-5"
  />
  <select bind:value={siteFilter} class={select} aria-label={t('users.filter_site')}>
    <option value="">{t('users.all_sites')}</option>
    {#each data.sites as s (s.id)}<option value={String(s.id)}>{s.name}</option>{/each}
  </select>
  <select bind:value={groupFilter} class={select} aria-label={t('users.filter_group')}>
    <option value="">{t('users.all_groups')}</option>
    <option value="admins">{t('users.admins')}</option>
    {#each data.groups as g (g.id)}<option value={String(g.id)}>{g.name}</option>{/each}
    <option value="none">{t('users.no_group')}</option>
  </select>
  <select bind:value={loginFilter} class={select} aria-label={t('users.filter_login')}>
    <option value="">{t('users.any_login')}</option>
    <option value="password">{t('users.login_password')}</option>
    <option value="discord">Discord</option>
    <option value="google">Google</option>
  </select>
  <select bind:value={statusFilter} class={select} aria-label={t('users.filter_status')}>
    {#each statusFilters as f}<option value={f.id}>{f.name}</option>{/each}
  </select>
  <select bind:value={sortBy} class={select} aria-label={t('users.sort')}>
    {#each sorts as s}<option value={s.id}>{t('users.sort_option', { name: s.name })}</option>{/each}
  </select>
</div>
