import { screen, waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, it, vi } from "vitest";
import * as m from "../../paraglide/messages";
import { AddRepoModal } from "./AddRepoModal";
import { renderWithProviders } from "../../test/render";
import { backrestService } from "../../api/client";
import { alerts } from "../../components/common/Alerts";
import { makeConfig, makeRepo, connectError, Code } from "../../test/proto";
import { CheckRepoExistsResponseSchema } from "../../../gen/ts/v1/service_pb";
import { ResticSnapshotListSchema } from "../../../gen/ts/v1/restic_pb";
import { StringListSchema } from "../../../gen/ts/types/value_pb";

// The URIAutocomplete combobox is the only <input role="combobox"> in the
// modal (the Advanced-section EnumSelectors render as <button> triggers).
const getUriInput = (): HTMLInputElement =>
  screen
    .getAllByRole("combobox")
    .find((el) => el.tagName === "INPUT") as HTMLInputElement;

const getPasswordInput = (): HTMLInputElement =>
  screen.getByLabelText(m.login_password_placeholder()) as HTMLInputElement;

// Fills the three required create-mode fields (name / uri / password).
const fillCreateForm = async (
  user: ReturnType<typeof renderWithProviders>["user"],
  { id, uri, password }: { id: string; uri: string; password: string },
) => {
  const nameInput = await screen.findByPlaceholderText("repo1");
  await user.type(nameInput, id);
  // The autocomplete combobox drops characters under per-key typing; paste the
  // whole URI in a single input event instead.
  await user.click(getUriInput());
  await user.paste(uri);
  await user.type(getPasswordInput(), password);
};

describe("AddRepoModal", () => {
  beforeEach(() => {
    // pathAutocomplete is called by the URIAutocomplete on every keystroke; it
    // must return a thenable StringList or the input handler throws.
    vi.mocked(backrestService.pathAutocomplete).mockResolvedValue(
      create(StringListSchema, { values: [] }),
    );
    vi.mocked(backrestService.listSnapshots).mockResolvedValue(
      create(ResticSnapshotListSchema, {}),
    );
  });

  it("renders name, uri, password fields and the submit button in create mode", async () => {
    renderWithProviders(<AddRepoModal template={null} />, {
      config: makeConfig(),
    });

    expect(await screen.findByPlaceholderText("repo1")).toBeInTheDocument();
    expect(getUriInput()).toBeInTheDocument();
    expect(getPasswordInput()).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: m.add_plan_modal_button_submit() }),
    ).toBeInTheDocument();
  });

  it("blocks submission and surfaces validation feedback for an invalid repo name", async () => {
    const errorSpy = vi.spyOn(alerts, "error");
    const { user } = renderWithProviders(<AddRepoModal template={null} />, {
      config: makeConfig(),
    });

    // "bad name" contains a space, which fails namePattern.
    const nameInput = await screen.findByPlaceholderText("repo1");
    await user.type(nameInput, "bad name");

    // Inline validation feedback becomes visible.
    expect(
      await screen.findByText(m.settings_auth_name_pattern()),
    ).toBeInTheDocument();

    await user.click(
      screen.getByRole("button", { name: m.add_plan_modal_button_submit() }),
    );

    expect(backrestService.addRepo).not.toHaveBeenCalled();
    expect(errorSpy).toHaveBeenCalled();
  });

  it("creates a repo: calls addRepo, updates config context, and shows a success toast", async () => {
    const successSpy = vi.spyOn(alerts, "success");
    const resolvedConfig = makeConfig({
      repos: [makeRepo({ id: "myrepo", uri: "/tmp/repo" })],
    });
    vi.mocked(backrestService.addRepo).mockResolvedValue(resolvedConfig);

    const { user, setConfig } = renderWithProviders(
      <AddRepoModal template={null} />,
      { config: makeConfig() },
    );

    await fillCreateForm(user, {
      id: "myrepo",
      uri: "/tmp/repo",
      password: "supersecret",
    });

    await user.click(
      screen.getByRole("button", { name: m.add_plan_modal_button_submit() }),
    );

    await waitFor(() => expect(backrestService.addRepo).toHaveBeenCalled());

    // The submit path (handleOk) writes directly via addRepo; checkRepoExists
    // is only exercised by the separate "Test Configuration" button.
    expect(backrestService.checkRepoExists).not.toHaveBeenCalled();
    expect(backrestService.addRepo).toHaveBeenCalledWith(
      expect.objectContaining({
        repo: expect.objectContaining({
          id: "myrepo",
          uri: "/tmp/repo",
          password: "supersecret",
        }),
      }),
    );
    expect(setConfig).toHaveBeenCalledWith(resolvedConfig);
    expect(successSpy).toHaveBeenCalledWith(
      m.add_repo_modal_success_added({ uri: "/tmp/repo" }),
    );
  });

  it("Test Configuration on an existing repo calls checkRepoExists and reports it exists", async () => {
    const successSpy = vi.spyOn(alerts, "success");
    vi.mocked(backrestService.checkRepoExists).mockResolvedValue(
      create(CheckRepoExistsResponseSchema, { exists: true }),
    );

    const { user } = renderWithProviders(<AddRepoModal template={null} />, {
      config: makeConfig(),
    });

    await fillCreateForm(user, {
      id: "myrepo",
      uri: "/tmp/repo",
      password: "supersecret",
    });

    await user.click(
      screen.getByRole("button", { name: m.add_repo_modal_test_config() }),
    );

    await waitFor(() =>
      expect(backrestService.checkRepoExists).toHaveBeenCalled(),
    );
    expect(backrestService.checkRepoExists).toHaveBeenCalledWith(
      expect.objectContaining({
        repo: expect.objectContaining({ id: "myrepo", uri: "/tmp/repo" }),
      }),
    );
    expect(backrestService.addRepo).not.toHaveBeenCalled();
    expect(successSpy).toHaveBeenCalledWith(
      m.add_repo_modal_test_success_existing({ uri: "/tmp/repo" }),
    );
  });

  it("edit mode prefills and disables identity fields and deletes via confirm", async () => {
    const template = makeRepo({
      id: "existing-repo",
      uri: "/tmp/existing",
      password: "existing-pw",
    });
    const resolvedConfig = makeConfig();
    vi.mocked(backrestService.removeRepo).mockResolvedValue(resolvedConfig);

    const { user, setConfig } = renderWithProviders(
      <AddRepoModal template={template} />,
      { config: makeConfig({ repos: [template] }) },
    );

    // Prefilled + disabled identity/connection fields.
    const nameInput = (await screen.findByDisplayValue(
      "existing-repo",
    )) as HTMLInputElement;
    expect(nameInput).toBeDisabled();
    expect(getUriInput()).toHaveValue("/tmp/existing");
    expect(getUriInput()).toBeDisabled();
    expect(getPasswordInput()).toBeDisabled();

    // Delete requires a confirm click (ConfirmButton second press).
    await user.click(
      screen.getByRole("button", { name: m.add_plan_modal_button_delete() }),
    );
    const confirm = await screen.findByRole("button", {
      name: m.add_plan_modal_button_confirm_delete(),
    });
    await user.click(confirm);

    await waitFor(() => expect(backrestService.removeRepo).toHaveBeenCalled());
    expect(backrestService.removeRepo).toHaveBeenCalledWith(
      expect.objectContaining({ repoId: "existing-repo" }),
    );
    expect(setConfig).toHaveBeenCalledWith(resolvedConfig);
  });

  it("uses onSaveOverride when provided and does not call addRepo", async () => {
    const successSpy = vi.spyOn(alerts, "success");
    const onSaveOverride = vi.fn().mockResolvedValue(undefined);

    const { user } = renderWithProviders(
      <AddRepoModal template={null} onSaveOverride={onSaveOverride} />,
      { config: makeConfig() },
    );

    await fillCreateForm(user, {
      id: "myrepo",
      uri: "/tmp/repo",
      password: "supersecret",
    });

    await user.click(
      screen.getByRole("button", { name: m.add_plan_modal_button_submit() }),
    );

    await waitFor(() => expect(onSaveOverride).toHaveBeenCalled());
    expect(onSaveOverride).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "myrepo",
        uri: "/tmp/repo",
        password: "supersecret",
      }),
    );
    expect(backrestService.addRepo).not.toHaveBeenCalled();
    expect(successSpy).toHaveBeenCalledWith(
      m.add_repo_modal_success_updated({ uri: "/tmp/repo" }),
    );
  });

  it("shows an error toast and keeps the modal open when addRepo rejects", async () => {
    const errorSpy = vi.spyOn(alerts, "error");
    vi.mocked(backrestService.addRepo).mockRejectedValue(
      connectError(Code.Internal, "init failed"),
    );

    const { user } = renderWithProviders(<AddRepoModal template={null} />, {
      config: makeConfig(),
    });

    await fillCreateForm(user, {
      id: "myrepo",
      uri: "/tmp/repo",
      password: "supersecret",
    });

    const submit = screen.getByRole("button", {
      name: m.add_plan_modal_button_submit(),
    });
    await user.click(submit);

    await waitFor(() => expect(errorSpy).toHaveBeenCalled());
    const [content] = errorSpy.mock.calls[errorSpy.mock.calls.length - 1];
    expect(String(content)).toContain("init failed");

    // Modal is still mounted and the submit button is interactive again.
    expect(
      screen.getByRole("button", { name: m.add_plan_modal_button_submit() }),
    ).not.toBeDisabled();
  });
});
