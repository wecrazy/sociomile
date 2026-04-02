import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import YAML from "yaml";

type Dictionary = Record<string, unknown>;

type I18nContextValue = {
  locale: string;
  setLocale: (locale: string) => void;
  t: (key: string, vars?: Record<string, string>) => string;
};

const STORAGE_KEY = "sociomile-locale";
const I18nContext = createContext<I18nContextValue | undefined>(undefined);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState(() => window.localStorage.getItem(STORAGE_KEY) ?? "en");
  const [dictionaries, setDictionaries] = useState<Record<string, Dictionary>>({});

  useEffect(() => {
    async function load(target: string) {
      const [english, current] = await Promise.all([
        fetchDictionary("en"),
        fetchDictionary(target),
      ]);

      setDictionaries({ en: english, [target]: current });
    }

    load(locale).catch(() => undefined);
  }, [locale]);

  const value = useMemo<I18nContextValue>(
    () => ({
      locale,
      setLocale: (nextLocale: string) => {
        window.localStorage.setItem(STORAGE_KEY, nextLocale);
        setLocaleState(nextLocale);
      },
      t: (key: string, vars?: Record<string, string>) =>
        translate(dictionaries[locale], dictionaries.en, key, vars),
    }),
    [dictionaries, locale],
  );

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error("useI18n must be used within I18nProvider");
  }

  return context;
}

async function fetchDictionary(locale: string): Promise<Dictionary> {
  const response = await fetch(`/locales/${locale}.yaml`);
  if (!response.ok) {
    return {};
  }

  const source = await response.text();
  return (YAML.parse(source) as Dictionary) ?? {};
}

function translate(
  dictionary: Dictionary | undefined,
  fallback: Dictionary | undefined,
  key: string,
  vars?: Record<string, string>,
) {
  const raw = pick(dictionary, key) ?? pick(fallback, key) ?? key;
  if (typeof raw !== "string") {
    return key;
  }

  return Object.entries(vars ?? {}).reduce(
    (result, [name, value]) => result.replaceAll(`{{${name}}}`, value),
    raw,
  );
}

function pick(dictionary: Dictionary | undefined, key: string): unknown {
  return key.split(".").reduce<unknown>((current, segment) => {
    if (current && typeof current === "object" && segment in current) {
      return (current as Dictionary)[segment];
    }

    return undefined;
  }, dictionary);
}
