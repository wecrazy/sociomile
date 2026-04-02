import { afterEach, describe, expect, it, vi } from "vitest";

const renderSpy = vi.fn();
const createRootSpy = vi.fn(() => ({ render: renderSpy }));

vi.mock("react-dom/client", () => ({
  default: {
    createRoot: createRootSpy,
  },
}));

describe("main", () => {
  afterEach(() => {
    document.body.innerHTML = "";
    renderSpy.mockClear();
    createRootSpy.mockClear();
    vi.resetModules();
  });

  it("mounts the application into the root element", async () => {
    document.body.innerHTML = '<div id="root"></div>';

    await import("./main");

    expect(createRootSpy).toHaveBeenCalledWith(document.getElementById("root"));
    expect(renderSpy).toHaveBeenCalledTimes(1);
  });
});
