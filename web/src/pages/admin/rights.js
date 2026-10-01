// What the signed-in staff member may do; mirrors store.Rights on the server (which enforces it).

export const permLabels = {
  approve: { name: 'Approve requests', hint: 'Approve or deny access requests for every site.' },
  users: { name: 'Manage users', hint: 'Create users, block, reset passwords, delete.' },
  groups: { name: 'Manage groups & rules', hint: 'Create groups, change their members and auto-add rules.' },
  sites: { name: 'Manage sites', hint: 'Add, change and remove sites and see the NPM config.' },
  settings: { name: 'Settings', hint: 'Email, Discord and login settings.' },
  appearance: { name: 'Appearance', hint: 'Logo, colors, font and background.' },
  audit: { name: 'View audit log', hint: 'See who did what in the admin panel.' },
};

export const can = (data, perm) => data.me.admin || data.me.perms.includes(perm);

export const canApprove = (data, siteId) => can(data, 'approve') || data.me.approve_sites.includes(siteId);

export const canApproveAny = (data) => can(data, 'approve') || data.me.approve_sites.length > 0;

// Which tabs this person gets.
export function visibleTabs(data) {
  const t = [];
  if (canApproveAny(data) || can(data, 'users')) t.push('requests');
  if (can(data, 'users') || can(data, 'groups')) t.push('users', 'groups');
  if (can(data, 'sites')) t.push('sites');
  if (can(data, 'appearance')) t.push('appearance');
  if (can(data, 'settings')) t.push('settings');
  if (can(data, 'audit')) t.push('audit');
  return t;
}

// A user's standing for one site, the same way the server decides access:
// 'admin' | 'needs email' | 'open' | 'approved' | 'denied' | 'group' | 'pending' | ''.
export function standing(data, u, site) {
  if (u.is_admin) return 'admin';
  if (site.require_email && !u.email_verified) return 'needs email';
  if (!site.require_approval) return 'open';
  const direct = data.access.find((a) => a.user_id === u.id && a.site_id === site.id)?.status;
  if (direct === 'approved' || direct === 'denied') return direct;
  if (groupsOpening(data, u, site).length > 0) return 'group';
  return direct || '';
}

export const opens = (st) => st === 'admin' || st === 'open' || st === 'approved' || st === 'group';

export const groupsOf = (data, u) => {
  const ids = new Set(data.memberships.filter((m) => m.user_id === u.id).map((m) => m.group_id));
  return data.groups.filter((g) => ids.has(g.id));
};

// The user's groups that may open the site.
export function groupsOpening(data, u, site) {
  const openers = new Set(data.group_sites.filter((gs) => gs.site_id === site.id && gs.can_open).map((gs) => gs.group_id));
  return groupsOf(data, u).filter((g) => openers.has(g.id));
}

export const standingText = {
  admin: 'admin',
  open: 'has access',
  approved: 'approved',
  group: 'via group',
  pending: 'pending',
  denied: 'denied',
  'needs email': 'needs verified email',
};

export const standingBadge = {
  admin: 'bg-indigo-500/15 text-indigo-300 ring-indigo-400/20',
  open: 'bg-emerald-500/15 text-emerald-300 ring-emerald-500/25',
  approved: 'bg-emerald-500/15 text-emerald-300 ring-emerald-500/25',
  group: 'bg-emerald-500/15 text-emerald-300 ring-emerald-500/25',
  pending: 'bg-amber-500/15 text-amber-300 ring-amber-500/25',
  denied: 'bg-rose-500/15 text-rose-300 ring-rose-500/25',
  'needs email': 'bg-sky-500/15 text-sky-300 ring-sky-500/25',
};
