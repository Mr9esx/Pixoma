import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import zhCN from "./locales/zh-CN";
import en from "./locales/en";

export const supportedLocales = ["zh-CN", "en"] as const;
export type SupportedLocale = (typeof supportedLocales)[number];

const resources = {
  "zh-CN": { translation: zhCN },
  en: { translation: en },
} as const;

export function resolveLocale(langSegment?: string): SupportedLocale {
  if (langSegment === "en") return "en";
  return "zh-CN";
}

export async function initI18n(lng: SupportedLocale = "zh-CN") {
  if (!i18n.isInitialized) {
    await i18n.use(initReactI18next).init({
      resources,
      lng,
      fallbackLng: "zh-CN",
      interpolation: { escapeValue: false },
    });
  }
  return i18n;
}

export default i18n;
