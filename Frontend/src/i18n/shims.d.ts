// Augment Vue's component instance types so `$t` / `$i18n` are recognized by
// vue-tsc during type checking.
import { SUPPORTED_LOCALES } from '@/i18n';

declare module 'vue' {
  interface ComponentCustomProperties {
    $t: (key: string, params?: Record<string, unknown>) => string;
    $i18n: {
      locale: string;
      setLocale: (locale: string) => void;
      availableLocales: typeof SUPPORTED_LOCALES;
    };
  }
}
