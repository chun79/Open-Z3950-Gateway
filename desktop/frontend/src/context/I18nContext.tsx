import React, { createContext, useContext, useState, ReactNode } from 'react';
import en from '../locales/en.json';
import zh from '../locales/zh.json';

type Locale = 'en' | 'zh';
type Translations = typeof en;

interface I18nContextType {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: keyof Translations, params?: Record<string, string>) => string;
}

const I18nContext = createContext<I18nContextType | undefined>(undefined);

const translations: Record<Locale, Translations> = { en, zh };

export function I18nProvider({ children }: { children: ReactNode }) {
  // Try to detect browser language or default to 'zh' given user preference
  const defaultLocale = navigator.language.startsWith('zh') ? 'zh' : 'en';
  const [locale, setLocale] = useState<Locale>(defaultLocale);

  const t = (key: keyof Translations, params?: Record<string, string>) => {
    let text = translations[locale][key] || key;
    if (params) {
      Object.entries(params).forEach(([k, v]) => {
        text = text.replace(`{${k}}`, v);
      });
    }
    return text;
  };

  return (
    <I18nContext.Provider value={{ locale, setLocale, t }}>
      {children}
    </I18nContext.Provider>
  );
}

export function useI18n() {
  const context = useContext(I18nContext);
  if (context === undefined) {
    throw new Error('useI18n must be used within an I18nProvider');
  }
  return context;
}
