import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { installFetchMock, localeRoutes } from "../test/mock-fetch";
import { renderWithProviders } from "../test/test-utils";
import { DataTable } from "./DataTable";

describe("DataTable", () => {
  it("renders loading and empty states with disabled pagination", async () => {
    installFetchMock(localeRoutes());

    const loadingView = renderWithProviders(
      <DataTable
        columns={[{ key: "name", header: "Name", render: (row: { name: string }) => row.name }]}
        rows={[]}
        total={0}
        offset={0}
        limit={10}
        loading={true}
        onPageChange={vi.fn()}
      />,
      { route: "/", path: "/" },
    );

    await waitFor(() =>
      expect(loadingView.container.querySelector(".table-empty-state")).toHaveTextContent(
        "Loading...",
      ),
    );
    expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();

    loadingView.unmount();

    const emptyView = renderWithProviders(
      <DataTable
        columns={[{ key: "name", header: "Name", render: (row: { name: string }) => row.name }]}
        rows={[]}
        total={0}
        offset={0}
        limit={10}
        loading={false}
        onPageChange={vi.fn()}
      />,
    );

    await waitFor(() =>
      expect(emptyView.container.querySelector(".table-empty-state")).toHaveTextContent(
        "No data available.",
      ),
    );
  });

  it("renders rows and invokes pagination callbacks", async () => {
    installFetchMock(localeRoutes());

    const onPageChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <DataTable
        columns={[
          { key: "name", header: "Name", render: (row: { name: string }) => row.name },
          { key: "value", header: "Value", render: (row: { value: string }) => row.value },
        ]}
        rows={[{ name: "Row 1", value: "Value 1" }]}
        total={25}
        offset={10}
        limit={10}
        loading={false}
        onPageChange={onPageChange}
      />,
      { route: "/", path: "/" },
    );

    expect(await screen.findByText("Showing 11 to 20 of 25")).toBeInTheDocument();
    expect(screen.getByText("Row 1")).toBeInTheDocument();
    expect(screen.getByText("Value 1")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Previous" }));
    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(onPageChange).toHaveBeenNthCalledWith(1, 0);
    expect(onPageChange).toHaveBeenNthCalledWith(2, 20);
  });
});
