// What the signed-in staff member may do; mirrors store.Rights on the server (which enforces it).

import { t } from '../../lib/i18n.svelte.js';

// Labels for each right. The getters translate on every read, so the text follows the
// current language: permLabels[perm].name / .hint.
const permEntry = (key) => ({
  get name() {
    return t(`rights.${key}`);
  },
  get hint() {
    return t(`rights.${key}_hint`);
  },
});

export const permLabels = {
  approve: permEntry('approve'),
  users: permEntry('users'),
  groups: permEntry('groups'),
  sites: permEntry('sites'),
  settings: permEntry('settings'),
  appearance: permEntry('appearance'),
  audit: permEntry('audit'),
};

export const can = (data, perm) => data.me.admin || data.me.perms.includes(perm);

export const canApprove = (data, siteId) => can(data, 'approve') || data.me.approve_sites.includes(siteId);

export const canApproveAny = (data) => can(data, 'approve') || data.me.approve_sites.length > 0;

// Which tabs this person gets.
export function visibleTabs(data) {
  const tabs = [];
  if (canApproveAny(data) || can(data, 'users')) tabs.push('requests');
  if (can(data, 'users') || can(data, 'groups')) tabs.push('users', 'groups');
  if (can(data, 'sites')) tabs.push('sites');
  if (can(data, 'appearance')) tabs.push('appearance');
  if (can(data, 'settings')) tabs.push('settings');
  if (can(data, 'audit')) tabs.push('audit');
  return tabs;
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

// standingText[st] gives the translated label; the getters read the current language.
const standingKeys = {
  admin: 'rights.standing_admin',
  open: 'rights.standing_open',
  approved: 'rights.standing_approved',
  group: 'rights.standing_group',
  pending: 'rights.standing_pending',
  denied: 'rights.standing_denied',
  'needs email': 'rights.standing_needs_email',
};
export const standingText = {};
for (const [st, key] of Object.entries(standingKeys)) {
  Object.defineProperty(standingText, st, { get: () => t(key), enumerable: true });
}

export const standingBadge = {
  admin: 'bg-indigo-500/15 text-indigo-300 ring-indigo-400/20',
  open: 'bg-emerald-500/15 text-emerald-300 ring-emerald-500/25',
  approved: 'bg-emerald-500/15 text-emerald-300 ring-emerald-500/25',
  group: 'bg-emerald-500/15 text-emerald-300 ring-emerald-500/25',
  pending: 'bg-amber-500/15 text-amber-300 ring-amber-500/25',
  denied: 'bg-rose-500/15 text-rose-300 ring-rose-500/25',
  'needs email': 'bg-sky-500/15 text-sky-300 ring-sky-500/25',
};
