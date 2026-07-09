import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { ModalContextProvider, useShowModal } from "./ModalManager";

// ModalContextProvider renders `model` (whatever showModal() was last called
// with) directly at the provider root - it's not a portal, but any consumer
// deep in the tree can trigger content to appear there via useShowModal().
const Probe = () => {
  const showModal = useShowModal();
  return (
    <div>
      <button
        onClick={() => showModal(<div data-testid="probe-modal">hi</div>)}
      >
        show
      </button>
      <button onClick={() => showModal(null)}>hide</button>
    </div>
  );
};

describe("ModalManager", () => {
  it("renders content passed to useShowModal()'s setter at the provider root", async () => {
    const user = userEvent.setup();
    render(
      <ModalContextProvider>
        <Probe />
      </ModalContextProvider>,
    );

    expect(screen.queryByTestId("probe-modal")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "show" }));
    expect(screen.getByTestId("probe-modal")).toBeInTheDocument();
  });

  it("clears the rendered content when showModal(null) is called", async () => {
    const user = userEvent.setup();
    render(
      <ModalContextProvider>
        <Probe />
      </ModalContextProvider>,
    );

    await user.click(screen.getByRole("button", { name: "show" }));
    expect(screen.getByTestId("probe-modal")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "hide" }));
    expect(screen.queryByTestId("probe-modal")).not.toBeInTheDocument();
  });
});
