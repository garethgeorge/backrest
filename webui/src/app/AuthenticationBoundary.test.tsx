import { act, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import * as m from "../paraglide/messages";
import { AuthenticationBoundary } from "./App";
import { renderWithProviders } from "../test/render";
import { backrestService } from "../api/client";
import {
  connectError,
  Code,
  makeConfig,
  makeFirstRunConfig,
} from "../test/proto";

// AuthenticationBoundary lazily `await import(...)`s the real SettingsModal,
// which pulls in the full settings feature. Replace it with a cheap sentinel so
// the "first run" branch is observable without mounting that subtree. The
// specifier must match the one used inside App.tsx ("../features/settings/
// SettingsModal"); vitest resolves it relative to this file, which shares
// src/app/ with App.tsx, so the same relative path targets the same module.
vi.mock("../features/settings/SettingsModal", () => ({
  SettingsModal: () => <div data-testid="settings-modal-sentinel" />,
}));

const Child = () => <div data-testid="content" />;

describe("AuthenticationBoundary", () => {
  it("renders children and stores the config when getConfig succeeds", async () => {
    const config = makeConfig();
    vi.mocked(backrestService.getConfig).mockResolvedValue(config);

    const { setConfig } = renderWithProviders(
      <AuthenticationBoundary>
        <Child />
      </AuthenticationBoundary>,
      { statefulConfig: true },
    );

    expect(await screen.findByTestId("content")).toBeInTheDocument();
    expect(setConfig).toHaveBeenCalledWith(config);
    expect(
      screen.queryByTestId("settings-modal-sentinel"),
    ).not.toBeInTheDocument();
    expect(screen.queryByText(m.login_title())).not.toBeInTheDocument();
  });

  it("auto-opens the settings modal for a first-run config", async () => {
    const config = makeFirstRunConfig();
    vi.mocked(backrestService.getConfig).mockResolvedValue(config);

    const { setConfig } = renderWithProviders(
      <AuthenticationBoundary>
        <Child />
      </AuthenticationBoundary>,
      { statefulConfig: true },
    );

    // shouldShowSettings(firstRunConfig) === true → SettingsModal is shown.
    expect(
      await screen.findByTestId("settings-modal-sentinel"),
    ).toBeInTheDocument();
    // The config is still stored, so (with statefulConfig) children also render.
    expect(setConfig).toHaveBeenCalledWith(config);
    expect(screen.getByTestId("content")).toBeInTheDocument();
  });

  it("shows the login modal and hides children on an Unauthenticated error", async () => {
    vi.mocked(backrestService.getConfig).mockRejectedValue(
      connectError(Code.Unauthenticated),
    );

    const { setConfig } = renderWithProviders(
      <AuthenticationBoundary>
        <Child />
      </AuthenticationBoundary>,
      { statefulConfig: true },
    );

    expect(await screen.findByText(m.login_title())).toBeInTheDocument();
    expect(setConfig).not.toHaveBeenCalled();
    expect(screen.queryByTestId("content")).not.toBeInTheDocument();
  });

  it("shows a failure state with a working retry on a non-transient error", async () => {
    const config = makeConfig();
    vi.mocked(backrestService.getConfig)
      .mockRejectedValueOnce(connectError(Code.InvalidArgument))
      .mockResolvedValue(config);

    const { user } = renderWithProviders(
      <AuthenticationBoundary>
        <Child />
      </AuthenticationBoundary>,
      { statefulConfig: true },
    );

    // Non-transient errors break out of the retry loop immediately.
    expect(
      await screen.findByText("Failed to load configuration"),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("content")).not.toBeInTheDocument();
    expect(backrestService.getConfig).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole("button", { name: "Retry" }));

    // Retry re-invokes getConfig and children appear.
    expect(await screen.findByTestId("content")).toBeInTheDocument();
    expect(backrestService.getConfig).toHaveBeenCalledTimes(2);
  });

  describe("with fake timers", () => {
    afterEach(() => {
      vi.useRealTimers();
    });

    it("retries transient failures with backoff, then renders children", async () => {
      vi.useFakeTimers();
      const config = makeConfig();
      vi.mocked(backrestService.getConfig)
        .mockRejectedValueOnce(connectError(Code.Unavailable))
        .mockRejectedValueOnce(connectError(Code.Unavailable))
        .mockResolvedValue(config);

      renderWithProviders(
        <AuthenticationBoundary>
          <Child />
        </AuthenticationBoundary>,
        { statefulConfig: true },
      );

      // Backoff is 1000 * 2**attempt: 1000ms after the first failure, 2000ms
      // after the second. Advancing ~3s covers both without ever reaching the
      // per-attempt 10s Promise.race timeout (which would fire an unhandled
      // rejection against the already-settled getConfig).
      await act(async () => {
        await vi.advanceTimersByTimeAsync(3001);
      });

      expect(screen.getByTestId("content")).toBeInTheDocument();
      // 3 attempts: fail, fail, succeed.
      expect(backrestService.getConfig).toHaveBeenCalledTimes(3);
    });
  });
});
