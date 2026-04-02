import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { installFetchMock, textResponse } from "../test/mock-fetch";
import { I18nProvider, useI18n } from "./i18n";

const englishLocale = `greeting:
  hello: Hello {{name}}
  onlyEn: English only
`;

const indonesianLocale = `greeting:
  hello: Halo {{name}}
`;

describe("I18nProvider", () => {
  it("falls back to English when the selected locale cannot be loaded", async () => {
    localStorage.setItem("sociomile-locale", "id");
    installFetchMock([
      { match: /\/locales\/en\.yaml$/, response: textResponse(englishLocale) },
      { match: /\/locales\/id\.yaml$/, response: new Response("missing", { status: 404 }) },
    ]);

    render(
      <I18nProvider>
        <I18nHarness />
      </I18nProvider>,
    );

    expect(await screen.findByTestId("hello")).toHaveTextContent("Hello Sam");
    expect(screen.getByTestId("fallback")).toHaveTextContent("English only");
    expect(screen.getByTestId("missing")).toHaveTextContent("missing.value");
  });

  it("switches locale and persists the selection", async () => {
    installFetchMock([
      { match: /\/locales\/en\.yaml$/, response: textResponse(englishLocale) },
      { match: /\/locales\/id\.yaml$/, response: textResponse(indonesianLocale) },
    ]);

    const user = userEvent.setup();
    render(
      <I18nProvider>
        <I18nHarness />
      </I18nProvider>,
    );

    expect(await screen.findByTestId("hello")).toHaveTextContent("Hello Sam");

    await user.click(screen.getByRole("button", { name: "Switch Locale" }));

    expect(await screen.findByTestId("hello")).toHaveTextContent("Halo Sam");
    expect(screen.getByTestId("fallback")).toHaveTextContent("English only");
    expect(localStorage.getItem("sociomile-locale")).toBe("id");
  });
});

function I18nHarness() {
  const { locale, setLocale, t } = useI18n();

  return (
    <div>
      <span data-testid="locale">{locale}</span>
      <span data-testid="hello">{t("greeting.hello", { name: "Sam" })}</span>
      <span data-testid="fallback">{t("greeting.onlyEn")}</span>
      <span data-testid="missing">{t("missing.value")}</span>
      <button onClick={() => setLocale("id")} type="button">
        Switch Locale
      </button>
    </div>
  );
}
