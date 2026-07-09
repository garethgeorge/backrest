import React, { useEffect } from "react";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import * as m from "../../paraglide/messages";
import { AddPlanModal } from "./AddPlanModal";
import { renderWithProviders } from "../../test/render";
import { makeConfig, makePlan, makeRepo } from "../../test/proto";
import { backrestService } from "../../api/client";
import { alerts } from "../../components/common/Alerts";
import { useShowModal } from "../../components/common/ModalManager";
import { StringListSchema } from "../../../gen/ts/types/value_pb";
import type { Config, Plan } from "../../../gen/ts/v1/config_pb";

// --- Harness -----------------------------------------------------------------
// AddPlanModal closes itself by calling showModal(null). Rendering it through the
// ModalManager (rather than directly) makes that call actually unmount the modal,
// so "modal closes" can be asserted via the title disappearing from the DOM.
const ModalHarness = (props: {
  template: Plan | null;
  onSaveOverride?: (plan: Plan) => Promise<void>;
}) => {
  const showModal = useShowModal();
  useEffect(() => {
    showModal(
      <AddPlanModal
        template={props.template}
        onSaveOverride={props.onSaveOverride}
      />,
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  return null;
};

const newUser = () => userEvent.setup({ pointerEventsCheck: 0 });

const renderModal = (opts: {
  config: Config;
  template?: Plan | null;
  onSaveOverride?: (plan: Plan) => Promise<void>;
}) =>
  renderWithProviders(
    <ModalHarness
      template={opts.template ?? null}
      onSaveOverride={opts.onSaveOverride}
    />,
    { config: opts.config },
  );

// --- Field helpers -----------------------------------------------------------
const nameInput = (config: Config): HTMLElement =>
  screen.getByPlaceholderText("plan" + (config.plans.length + 1));

const typeName = async (
  user: ReturnType<typeof newUser>,
  config: Config,
  value: string,
) => {
  const input = nameInput(config);
  await user.clear(input);
  await user.type(input, value);
};

const selectRepo = async (user: ReturnType<typeof newUser>, repoId = "r1") => {
  const placeholder = screen.getByText(
    m.add_plan_modal_field_repository_select(),
  );
  const trigger = placeholder.closest("button");
  expect(trigger).not.toBeNull();
  await user.click(trigger!);
  const option = await screen.findByRole("option", { name: repoId });
  await user.click(option);
};

const addPath = async (user: ReturnType<typeof newUser>, path: string) => {
  // The first "Add" button belongs to the (first) paths DynamicList.
  const addButtons = screen.getAllByRole("button", {
    name: m.add_plan_modal_field_add(),
  });
  await user.click(addButtons[0]);
  const input = await screen.findByPlaceholderText(
    m.add_plan_modal_field_paths(),
  );
  // Paste rather than type: the URIAutocomplete is a controlled combobox and
  // per-keystroke typing races with its re-render, dropping characters.
  input.focus();
  await user.paste(path);
  await waitFor(() =>
    expect(
      screen.getByPlaceholderText(m.add_plan_modal_field_paths()),
    ).toHaveValue(path),
  );
};

const submit = async (user: ReturnType<typeof newUser>) => {
  const button = screen.getByRole("button", {
    name: m.add_plan_modal_button_submit(),
  });
  await user.click(button);
};

const configWithRepo = (extra?: Parameters<typeof makeConfig>[0]) =>
  makeConfig({
    repos: [makeRepo({ id: "r1", guid: "g1" })],
    ...extra,
  });

describe("AddPlanModal", () => {
  beforeEach(() => {
    // The paths field uses URIAutocomplete which calls pathAutocomplete while
    // typing; give it a resolving stub so keystrokes don't throw.
    vi.mocked(backrestService.pathAutocomplete).mockResolvedValue(
      create(StringListSchema, {}),
    );
  });

  it("renders the create-mode form (name, repo select, paths, submit)", async () => {
    const config = configWithRepo();
    renderModal({ config });

    expect(
      await screen.findByText(m.add_plan_modal_title_add()),
    ).toBeInTheDocument();
    // Plan name input (placeholder derives from existing plan count).
    expect(screen.getByPlaceholderText("plan1")).toBeInTheDocument();
    // Repo select shows its placeholder before a choice is made.
    expect(
      screen.getByText(m.add_plan_modal_field_repository_select()),
    ).toBeInTheDocument();
    // Paths section label.
    expect(
      screen.getByText(m.add_plan_modal_field_paths()),
    ).toBeInTheDocument();
    // Submit button.
    expect(
      screen.getByRole("button", { name: m.add_plan_modal_button_submit() }),
    ).toBeInTheDocument();
  });

  it("lets the user pick a repository from the (portalled) select", async () => {
    const config = configWithRepo();
    renderModal({ config });
    const user = newUser();

    await screen.findByText(m.add_plan_modal_title_add());
    // Before selection, the placeholder is visible.
    expect(
      screen.getByText(m.add_plan_modal_field_repository_select()),
    ).toBeInTheDocument();

    await selectRepo(user, "r1");

    // After selection the placeholder is replaced by the chosen repo id.
    await waitFor(() => {
      expect(
        screen.queryByText(m.add_plan_modal_field_repository_select()),
      ).not.toBeInTheDocument();
    });
    expect(screen.getAllByText("r1").length).toBeGreaterThan(0);
  });

  it("blocks submit for an invalid plan id (namePattern) and never calls setConfig", async () => {
    const config = configWithRepo();
    const { setConfig } = renderModal({ config });
    const user = newUser();
    const errorSpy = vi.spyOn(alerts, "error");

    await screen.findByText(m.add_plan_modal_title_add());
    await typeName(user, config, "bad name"); // space fails namePattern
    await submit(user);

    await waitFor(() => expect(errorSpy).toHaveBeenCalled());
    const [content] = errorSpy.mock.calls[errorSpy.mock.calls.length - 1];
    expect(String(content)).toContain(
      m.add_plan_modal_validation_plan_name_pattern(),
    );
    expect(backrestService.setConfig).not.toHaveBeenCalled();
    expect(setConfig).not.toHaveBeenCalled();
  });

  it("blocks submit for a duplicate plan id already in the config", async () => {
    const config = configWithRepo({
      plans: [makePlan({ id: "dup", repo: "r1" })],
    });
    renderModal({ config });
    const user = newUser();
    const errorSpy = vi.spyOn(alerts, "error");

    await screen.findByText(m.add_plan_modal_title_add());
    await typeName(user, config, "dup");
    await submit(user);

    await waitFor(() => expect(errorSpy).toHaveBeenCalled());
    const [content] = errorSpy.mock.calls[errorSpy.mock.calls.length - 1];
    expect(String(content)).toContain(
      m.add_plan_modal_validation_plan_exists(),
    );
    expect(backrestService.setConfig).not.toHaveBeenCalled();
  });

  it("blocks submit when no repository is selected", async () => {
    const config = configWithRepo();
    renderModal({ config });
    const user = newUser();
    const errorSpy = vi.spyOn(alerts, "error");

    await screen.findByText(m.add_plan_modal_title_add());
    await typeName(user, config, "valid-plan");
    await submit(user); // repo left empty

    await waitFor(() => expect(errorSpy).toHaveBeenCalled());
    const [content] = errorSpy.mock.calls[errorSpy.mock.calls.length - 1];
    expect(String(content)).toContain(
      m.add_plan_modal_validation_repository_required(),
    );
    expect(backrestService.setConfig).not.toHaveBeenCalled();
  });

  it("creates a plan: appends to config.plans, saves, and closes", async () => {
    const config = configWithRepo();
    vi.mocked(backrestService.setConfig).mockImplementation(
      async (c: any) => c,
    );
    const { setConfig } = renderModal({ config });
    const user = newUser();

    await screen.findByText(m.add_plan_modal_title_add());
    await typeName(user, config, "my-plan");
    await selectRepo(user, "r1");
    await addPath(user, "/foo");
    await submit(user);

    await waitFor(() =>
      expect(backrestService.setConfig).toHaveBeenCalledTimes(1),
    );
    const savedConfig = vi.mocked(backrestService.setConfig).mock
      .calls[0][0] as Config;
    expect(savedConfig.plans).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          id: "my-plan",
          repo: "r1",
          paths: ["/foo"],
        }),
      ]),
    );
    // Context setConfig receives the resolved (server) config.
    expect(setConfig).toHaveBeenCalledWith(savedConfig);
    // Modal closes on success.
    await waitFor(() =>
      expect(
        screen.queryByText(m.add_plan_modal_title_add()),
      ).not.toBeInTheDocument(),
    );
  });

  it("edit mode: prefills fields, keeps id immutable, and replaces (not appends) the plan", async () => {
    const plan = makePlan({ id: "p1", repo: "r1", paths: ["/existing"] });
    const config = configWithRepo({ plans: [plan] });
    vi.mocked(backrestService.setConfig).mockImplementation(
      async (c: any) => c,
    );
    const { setConfig } = renderModal({ config, template: plan });
    const user = newUser();

    // Update-mode title + prefilled, disabled id input.
    expect(
      await screen.findByText(m.add_plan_modal_title_update()),
    ).toBeInTheDocument();
    const idInput = screen.getByDisplayValue("p1");
    expect(idInput).toBeDisabled();
    // Repo shown as the prefilled value.
    expect(screen.getAllByText("r1").length).toBeGreaterThan(0);

    await submit(user);

    await waitFor(() =>
      expect(backrestService.setConfig).toHaveBeenCalledTimes(1),
    );
    const savedConfig = vi.mocked(backrestService.setConfig).mock
      .calls[0][0] as Config;
    // Replaced in place: length unchanged, same id present.
    expect(savedConfig.plans).toHaveLength(1);
    expect(savedConfig.plans[0].id).toBe("p1");
    expect(setConfig).toHaveBeenCalledWith(savedConfig);
  });

  it("calls onSaveOverride with the built plan and does NOT call backrestService.setConfig", async () => {
    const config = configWithRepo();
    const onSaveOverride = vi.fn(async () => {});
    renderModal({ config, onSaveOverride });
    const user = newUser();

    await screen.findByText(m.add_plan_modal_title_add());
    await typeName(user, config, "ov-plan");
    await selectRepo(user, "r1");
    await addPath(user, "/bar");
    await submit(user);

    await waitFor(() => expect(onSaveOverride).toHaveBeenCalledTimes(1));
    expect(onSaveOverride).toHaveBeenCalledWith(
      expect.objectContaining({ id: "ov-plan", repo: "r1", paths: ["/bar"] }),
    );
    expect(backrestService.setConfig).not.toHaveBeenCalled();
    // Modal still closes after the override resolves.
    await waitFor(() =>
      expect(
        screen.queryByText(m.add_plan_modal_title_add()),
      ).not.toBeInTheDocument(),
    );
  });

  it("surfaces an error and keeps the modal open when setConfig rejects", async () => {
    const config = configWithRepo();
    vi.mocked(backrestService.setConfig).mockRejectedValue(
      new Error("save boom"),
    );
    const { setConfig } = renderModal({ config });
    const user = newUser();
    const errorSpy = vi.spyOn(alerts, "error");

    await screen.findByText(m.add_plan_modal_title_add());
    await typeName(user, config, "my-plan");
    await selectRepo(user, "r1");
    await addPath(user, "/foo");
    await submit(user);

    await waitFor(() => expect(errorSpy).toHaveBeenCalled());
    // Backend was called, but the context config was never updated...
    expect(backrestService.setConfig).toHaveBeenCalledTimes(1);
    expect(setConfig).not.toHaveBeenCalled();
    // ...and the modal remains open.
    expect(screen.getByText(m.add_plan_modal_title_add())).toBeInTheDocument();
  });
});
