// Same rules as the server (internal/auth/validate.go).
export function passwordRules(pw) {
  return [
    { ok: [...pw].length >= 6, text: 'At least 6 characters' },
    { ok: /\p{Lu}/u.test(pw), text: '1 capital letter' },
    { ok: /[^\p{L}\p{N}\s]/u.test(pw), text: '1 symbol, like ! @ # ?' },
  ];
}
