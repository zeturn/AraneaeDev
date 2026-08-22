import type { App, Plugin, Ref } from 'vue';

export interface LocaleOption {
  code: string;
  label: string;
}

export const SUPPORTED_LOCALES: readonly LocaleOption[];
export const currentLocale: Ref<string>;
export function setLocale(locale: string): void;
export function translate(key: string, params?: Record<string, unknown>): string;
export function useI18n(): {
  t: (key: string, params?: Record<string, unknown>) => string;
  locale: Ref<string>;
  setLocale: (locale: string) => void;
  availableLocales: readonly LocaleOption[];
};
export function createI18n(): Plugin & { install(app: App): void };

declare const i18n: {
  currentLocale: typeof currentLocale;
  setLocale: typeof setLocale;
  translate: typeof translate;
  useI18n: typeof useI18n;
  createI18n: typeof createI18n;
};

export default i18n;
