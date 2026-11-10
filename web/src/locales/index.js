// Every language the portal offers. Keep in sync with Supported in internal/i18n/i18n.go.
// Adding one: put <code>.json next to en.json (same keys), add a line here and in i18n.go,
// and add internal/i18n/mail/<code>.json for the mails.
export const languages = [
  { code: 'en', name: 'English', english: 'English' },
  { code: 'nl', name: 'Nederlands', english: 'Dutch' },
  { code: 'tr', name: 'Türkçe', english: 'Turkish' },
  { code: 'de', name: 'Deutsch', english: 'German' },
  { code: 'fr', name: 'Français', english: 'French' },
  { code: 'es', name: 'Español', english: 'Spanish' },
  { code: 'pt', name: 'Português', english: 'Portuguese' },
  { code: 'it', name: 'Italiano', english: 'Italian' },
  { code: 'pl', name: 'Polski', english: 'Polish' },
  { code: 'ru', name: 'Русский', english: 'Russian' },
  { code: 'uk', name: 'Українська', english: 'Ukrainian' },
  { code: 'ar', name: 'العربية', english: 'Arabic', rtl: true },
  { code: 'fa', name: 'فارسی', english: 'Persian', rtl: true },
  { code: 'hi', name: 'हिन्दी', english: 'Hindi' },
  { code: 'ja', name: '日本語', english: 'Japanese' },
  { code: 'ko', name: '한국어', english: 'Korean' },
  { code: 'zh-CN', name: '简体中文', english: 'Chinese (Simplified)' },
  { code: 'zh-TW', name: '繁體中文', english: 'Chinese (Traditional)' },
  { code: 'id', name: 'Bahasa Indonesia', english: 'Indonesian' },
  { code: 'vi', name: 'Tiếng Việt', english: 'Vietnamese' },
  { code: 'th', name: 'ไทย', english: 'Thai' },
  { code: 'sv', name: 'Svenska', english: 'Swedish' },
  { code: 'da', name: 'Dansk', english: 'Danish' },
  { code: 'no', name: 'Norsk', english: 'Norwegian' },
  { code: 'fi', name: 'Suomi', english: 'Finnish' },
];

export const byCode = Object.fromEntries(languages.map((l) => [l.code, l]));
