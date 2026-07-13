/*
 * Lightweight i18n engine for Araneae frontend.
 *
 * Mirrors the vue-i18n API surface used across the app:
 *   - useI18n()  -> { t, locale, setLocale, availableLocales }
 *   - global $t / $i18n available on every component instance
 *
 * No external dependency is required so the project builds offline.
 */

import { ref } from 'vue';
import zhCN from './messages/zh-CN';
import enUS from './messages/en-US';

export const SUPPORTED_LOCALES = [
  { code: 'zh-CN', label: '简体中文' },
  { code: 'en-US', label: 'English' },
];

const messages = {
  'zh-CN': zhCN,
  'en-US': enUS,
};

const STORAGE_KEY = 'araneae-locale';
const FALLBACK_LOCALE = 'zh-CN';

function detectLocale() {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved && messages[saved]) return saved;
  } catch (e) {
    /* ignore */
  }
  try {
    const nav = navigator.language || 'zh-CN';
    const lower = nav.toLowerCase();
    if (lower.startsWith('zh')) return 'zh-CN';
    if (lower.startsWith('en')) return 'en-US';
  } catch (e) {
    /* ignore */
  }
  return FALLBACK_LOCALE;
}

// Reactive current locale. Reading `.value` inside a render/effect tracks it,
// so any `t(...)` call re-evaluates when the user switches language.
export const currentLocale = ref(detectLocale());

export function setLocale(locale) {
  if (!messages[locale]) return;
  currentLocale.value = locale;
  try {
    localStorage.setItem(STORAGE_KEY, locale);
  } catch (e) {
    /* ignore */
  }
  try {
    document.documentElement.setAttribute('lang', locale);
  } catch (e) {
    /* ignore */
  }
}

function interpolate(template, params) {
  if (!params) return template;
  return template.replace(/\{(\w+)\}/g, (_, name) =>
    params[name] !== undefined ? String(params[name]) : `{${name}}`
  );
}

export function translate(key, params) {
  const locale = currentLocale.value;
  let value = messages[locale] ? messages[locale][key] : undefined;
  if (value === undefined && messages[FALLBACK_LOCALE]) {
    value = messages[FALLBACK_LOCALE][key];
  }
  if (value === undefined) {
    // fall back to the source key so the UI never breaks
    value = key;
  }
  if (typeof value !== 'string') return value;
  return interpolate(value, params);
}

export function useI18n() {
  return {
    t: (key, params) => translate(key, params),
    locale: currentLocale,
    setLocale,
    availableLocales: SUPPORTED_LOCALES,
  };
}

export function createI18n() {
  return {
    install(app) {
      app.config.globalProperties.$t = (key, params) => translate(key, params);
      app.config.globalProperties.$i18n = {
        get locale() {
          return currentLocale.value;
        },
        setLocale,
        availableLocales: SUPPORTED_LOCALES,
      };
      try {
        document.documentElement.setAttribute('lang', currentLocale.value);
      } catch (e) {
        /* ignore */
      }
    },
  };
}

export default { currentLocale, setLocale, translate, useI18n, createI18n };
