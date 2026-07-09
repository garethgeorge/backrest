import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ConfirmButton, SpinButton } from "./SpinButton";
import { renderWithProviders } from "../../test/render";

// NOTE: these tests deliberately use real timers (renderWithProviders +
// RTL's real-timer `waitFor`) rather than renderWithFakeTimerUser for any
// interaction that leaves onClickAsync's promise pending across the click:
// once SpinButton's onClick sets loading=true synchronously, the button's
// `disabled` attribute flips mid dispatch, and userEvent's click() promise
// never resolves under fake timers in that case (verified by isolated
// reproduction: the handler runs to completion and even fully settles
// loading back to false, yet the outer `await user.click(...)` still hangs
// past an 8s timeout - almost certainly React's own internal scheduler using
// a real setTimeout/MessageChannel that vi's fake clock intercepts but
// nothing in the test explicitly advances). Fake timers are only used below
// for the ConfirmButton auto-revert test, where the only click involved
// (the "arm" click) never awaits onClickAsync and so never disables the
// button mid-click.

describe("SpinButton", () => {
  it("shows a disabled/loading state while the async handler is pending, then settles", async () => {
    let resolveClick: () => void = () => {};
    const onClickAsync = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveClick = resolve;
        }),
    );

    const { user } = renderWithProviders(
      <SpinButton onClickAsync={onClickAsync}>Do Thing</SpinButton>,
    );

    const button = screen.getByRole("button", { name: "Do Thing" });
    expect(button).not.toBeDisabled();

    await user.click(button);

    expect(onClickAsync).toHaveBeenCalledTimes(1);
    expect(button).toBeDisabled();

    resolveClick();
    await waitFor(() => expect(button).not.toBeDisabled());
  });

  it("ignores additional clicks while already loading (button is disabled)", async () => {
    let resolveClick: () => void = () => {};
    const onClickAsync = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveClick = resolve;
        }),
    );

    const { user } = renderWithProviders(
      <SpinButton onClickAsync={onClickAsync}>Do Thing</SpinButton>,
    );

    const button = screen.getByRole("button", { name: "Do Thing" });
    await user.click(button);
    expect(button).toBeDisabled();

    // A disabled button doesn't dispatch click events in the browser (and
    // userEvent mirrors that), so this exercises the same re-entrancy
    // guard as the `if (loading) return;` check in SpinButton's onClick.
    await user.click(button);
    expect(onClickAsync).toHaveBeenCalledTimes(1);

    resolveClick();
    await waitFor(() => expect(button).not.toBeDisabled());
  });
});

describe("ConfirmButton", () => {
  it("requires two clicks: the first arms/changes the label, the second invokes the callback", async () => {
    const onClickAsync = vi.fn().mockResolvedValue(undefined);

    const { user } = renderWithProviders(
      <ConfirmButton onClickAsync={onClickAsync} confirmTitle="Really?">
        Delete
      </ConfirmButton>,
    );

    const button = screen.getByRole("button", { name: "Delete" });
    await user.click(button);

    expect(onClickAsync).not.toHaveBeenCalled();
    const confirmButton = screen.getByRole("button", { name: "Really?" });
    expect(confirmButton).toBeInTheDocument();

    await user.click(confirmButton);

    expect(onClickAsync).toHaveBeenCalledTimes(1);
  });

  it("reverts to the unconfirmed label after confirmTimeout elapses without a second click", async () => {
    // Real timers + a short confirmTimeout: ConfirmButton's "arm" click still
    // routes through SpinButton's onClick (awaiting the wrapped handler,
    // which briefly flips loading true/false even though it does no real
    // async work), which is enough to hit the fake-timer/act() hang
    // described above - so this uses real time instead of vi.useFakeTimers().
    const onClickAsync = vi.fn().mockResolvedValue(undefined);

    const { user } = renderWithProviders(
      <ConfirmButton
        onClickAsync={onClickAsync}
        confirmTitle="Really?"
        confirmTimeout={50}
      >
        Delete
      </ConfirmButton>,
    );

    const button = screen.getByRole("button", { name: "Delete" });
    await user.click(button);
    expect(screen.getByRole("button", { name: "Really?" })).toBeInTheDocument();

    await waitFor(
      () =>
        expect(
          screen.getByRole("button", { name: "Delete" }),
        ).toBeInTheDocument(),
      { timeout: 2000 },
    );
    expect(onClickAsync).not.toHaveBeenCalled();
  });
});
