import { fireEvent, screen, waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { describe, expect, it, vi } from "vitest";
import * as m from "../../paraglide/messages";
import { SettingsModal } from "./SettingsModal";
import { renderWithProviders } from "../../test/render";
import {
  makeConfig,
  makeFirstRunConfig,
  connectError,
  Code,
} from "../../test/proto";
import { backrestService, authenticationService } from "../../api/client";
import { alerts } from "../../components/common/Alerts";
import { StringValueSchema } from "../../../gen/ts/types/value_pb";

// The modal renders inside a Portal (TwoPaneModal -> DialogRoot/Portal), so the
// content lives on document.body; `screen` queries reach it. There is no
// getConfig-on-mount call: SettingsModal seeds its form from the injected
// ConfigContext value and only hits backrestService on save / token actions.
// generatePairingToken is only invoked from an explicit button, never on
// render, but default it to resolve harmlessly in case anything triggers it.
const primeDefaults = () => {
  vi.mocked(backrestService.generatePairingToken).mockResolvedValue({} as any);
};

const instancePlaceholder = m.settings_field_instance_id_placeholder();
// This message carries a trailing space; trim so exact text matching works.
const initialSetupTitle = m.settings_initial_setup_title().trim();

const getSaveButton = () =>
  screen.getByRole("button", { name: /save changes/i });

describe("SettingsModal (first-run path)", () => {
  it("renders the initial-setup prompt with an empty, editable instance field", async () => {
    primeDefaults();
    renderWithProviders(<SettingsModal />, { config: makeFirstRunConfig() });

    expect(await screen.findByText(initialSetupTitle)).toBeInTheDocument();
    expect(screen.getByText(m.app_menu_settings())).toBeInTheDocument();

    const instanceInput = screen.getByPlaceholderText(instancePlaceholder);
    expect(instanceInput).toHaveValue("");
    expect(instanceInput).toBeEnabled();
  });

  it("keeps Save disabled on a pristine first-run form and never calls setConfig", async () => {
    // NOTE: SettingsModal has no client-side instance-id *format* validation
    // (a bad id like "bad name!" is passed through and rejected server-side).
    // The concrete client-side guard against an accidental empty submit is the
    // dirty-gated Save button, which we assert here.
    primeDefaults();
    renderWithProviders(<SettingsModal />, { config: makeFirstRunConfig() });

    const saveButton = await screen.findByRole("button", {
      name: /save changes/i,
    });
    expect(saveButton).toBeDisabled();
    expect(backrestService.setConfig).not.toHaveBeenCalled();
  });

  it("saves with authentication disabled and does not hash any password", async () => {
    primeDefaults();
    vi.mocked(backrestService.setConfig).mockResolvedValue(
      makeConfig({ instance: "my-instance", auth: { disabled: true } }),
    );

    const { user } = renderWithProviders(<SettingsModal />, {
      config: makeFirstRunConfig(),
    });

    const instanceInput =
      await screen.findByPlaceholderText(instancePlaceholder);
    await user.type(instanceInput, "my-instance");

    // The disable-auth control is a hidden native checkbox rendered by
    // ToggleField; click it directly (the visible label wraps it).
    const disableSwitch = document.querySelector(
      'input[type="checkbox"]',
    ) as HTMLInputElement;
    expect(disableSwitch).toBeTruthy();
    fireEvent.click(disableSwitch);

    await waitFor(() => expect(getSaveButton()).toBeEnabled());
    await user.click(getSaveButton());

    await waitFor(() => {
      expect(backrestService.setConfig).toHaveBeenCalledWith(
        expect.objectContaining({
          instance: "my-instance",
          auth: expect.objectContaining({ disabled: true }),
        }),
      );
    });
    expect(authenticationService.hashPassword).not.toHaveBeenCalled();
  });

  it("hashes a new user's password and saves the hashed credential", async () => {
    primeDefaults();
    vi.mocked(authenticationService.hashPassword).mockResolvedValue(
      create(StringValueSchema, { value: "hashed-secret" }),
    );
    vi.mocked(backrestService.setConfig).mockResolvedValue(
      makeConfig({ instance: "my-instance" }),
    );

    const { user } = renderWithProviders(<SettingsModal />, {
      config: makeFirstRunConfig(),
    });

    const instanceInput =
      await screen.findByPlaceholderText(instancePlaceholder);
    await user.type(instanceInput, "my-instance");

    await user.click(
      screen.getByRole("button", { name: m.settings_auth_add_user() }),
    );

    const usernameInput = await screen.findByPlaceholderText(
      m.login_username_placeholder(),
    );
    const passwordInput = screen.getByPlaceholderText(
      m.login_password_placeholder(),
    );
    await user.type(usernameInput, "alice");
    await user.type(passwordInput, "plaintext-pw");

    await waitFor(() => expect(getSaveButton()).toBeEnabled());
    await user.click(getSaveButton());

    await waitFor(() => {
      expect(authenticationService.hashPassword).toHaveBeenCalledWith(
        expect.objectContaining({ value: "plaintext-pw" }),
      );
    });
    expect(backrestService.setConfig).toHaveBeenCalledWith(
      expect.objectContaining({
        instance: "my-instance",
        auth: expect.objectContaining({
          users: expect.arrayContaining([
            expect.objectContaining({
              name: "alice",
              password: expect.objectContaining({
                case: "passwordBcrypt",
                value: "hashed-secret",
              }),
            }),
          ]),
        }),
      }),
    );
  });

  it("shows an error toast and keeps the modal open when setConfig is rejected", async () => {
    primeDefaults();
    const errorSpy = vi.spyOn(alerts, "error");
    vi.mocked(backrestService.setConfig).mockRejectedValue(
      connectError(Code.InvalidArgument, "boom"),
    );

    const { user } = renderWithProviders(<SettingsModal />, {
      config: makeFirstRunConfig(),
    });

    const instanceInput =
      await screen.findByPlaceholderText(instancePlaceholder);
    await user.type(instanceInput, "my-instance");

    const disableSwitch = document.querySelector(
      'input[type="checkbox"]',
    ) as HTMLInputElement;
    fireEvent.click(disableSwitch);

    await waitFor(() => expect(getSaveButton()).toBeEnabled());
    await user.click(getSaveButton());

    await waitFor(() => expect(errorSpy).toHaveBeenCalled());
    const [content] = errorSpy.mock.calls[errorSpy.mock.calls.length - 1];
    expect(String(content)).toContain("boom");
    // Modal is still mounted: its title remains visible.
    expect(screen.getByText(m.app_menu_settings())).toBeInTheDocument();
  });

  it("renders a configured instance with an immutable instance field", async () => {
    primeDefaults();
    renderWithProviders(<SettingsModal />, { config: makeConfig() });

    const instanceInput =
      await screen.findByPlaceholderText(instancePlaceholder);
    expect(instanceInput).toHaveValue("test-instance");
    expect(instanceInput).toBeDisabled();
    // A configured instance is past first-run: no initial-setup prompt.
    expect(screen.queryByText(initialSetupTitle)).not.toBeInTheDocument();
  });
});
