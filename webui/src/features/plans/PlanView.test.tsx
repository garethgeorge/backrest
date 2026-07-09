import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import * as m from "../../paraglide/messages";
import { PlanView } from "./PlanView";
import { renderWithProviders } from "../../test/render";
import {
  makeConfig,
  makeRepo,
  makePlan,
  connectError,
  Code,
} from "../../test/proto";
import { backrestService } from "../../api/client";
import { alerts } from "../../components/common/Alerts";
import { DoRepoTaskRequest_Task } from "../../../gen/ts/v1/service_pb";

// The tree/list views pull in Chakra TreeView/Splitter machinery that isn't
// relevant to the action-button behavior under test here, and jsdom lacks
// several of the DOM primitives Zag.js leans on for those widgets. Stub them
// to trivial placeholders per the harness's allowance for mocking modules
// from within the test file (source is not edited).
vi.mock("../operations/OperationListView", () => ({
  OperationListView: () => <div data-testid="operation-list-view-stub" />,
}));
vi.mock("../operations/OperationTreeView", () => ({
  OperationTreeView: () => <div data-testid="operation-tree-view-stub" />,
}));

const config = makeConfig({
  repos: [makeRepo({ id: "test-repo", guid: "test-repo-guid" })],
  plans: [makePlan({ id: "test-plan", repo: "test-repo" })],
});
const plan = config.plans[0];

const openMenu = async (
  user: ReturnType<typeof renderWithProviders>["user"],
) => {
  const trigger = await screen.findByRole("button", { name: "More actions" });
  await user.click(trigger);
};

describe("PlanView", () => {
  it("renders the plan heading and backup button", async () => {
    renderWithProviders(<PlanView plan={plan} />, { config });

    expect(await screen.findByText("test-plan")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: m.plan_button_backup() }),
    ).toBeInTheDocument();
  });

  it("triggers a backup and shows a success toast when Backup Now is clicked", async () => {
    vi.mocked(backrestService.backup).mockResolvedValue({} as any);
    const successSpy = vi.spyOn(alerts, "success");

    const { user } = renderWithProviders(<PlanView plan={plan} />, { config });

    const backupButton = await screen.findByRole("button", {
      name: m.plan_button_backup(),
    });
    await user.click(backupButton);

    await waitFor(() => {
      expect(backrestService.backup).toHaveBeenCalledWith(
        expect.objectContaining({ value: "test-plan" }),
      );
    });
    expect(successSpy).toHaveBeenCalledWith(m.plan_backup_scheduled());
  });

  it("shows an error toast when the backup call is rejected", async () => {
    vi.mocked(backrestService.backup).mockRejectedValue(
      connectError(Code.Internal, "boom"),
    );
    const errorSpy = vi.spyOn(alerts, "error");

    const { user } = renderWithProviders(<PlanView plan={plan} />, { config });

    const backupButton = await screen.findByRole("button", {
      name: m.plan_button_backup(),
    });
    await user.click(backupButton);

    await waitFor(() => expect(errorSpy).toHaveBeenCalled());
    const [content] = errorSpy.mock.calls[errorSpy.mock.calls.length - 1];
    expect(String(content)).toContain(m.plan_error_backup());
  });

  it("triggers a dry-run backup from the menu with dryRun set", async () => {
    vi.mocked(backrestService.backup).mockResolvedValue({} as any);
    const successSpy = vi.spyOn(alerts, "success");

    const { user } = renderWithProviders(<PlanView plan={plan} />, { config });

    await openMenu(user);
    const dryRunItem = await screen.findByText(m.op_type_dry_run_backup());
    await user.click(dryRunItem);

    await waitFor(() => {
      expect(backrestService.backup).toHaveBeenCalledWith(
        expect.objectContaining({ value: "test-plan", dryRun: true }),
      );
    });
    expect(successSpy).toHaveBeenCalledWith(m.plan_dry_run_scheduled());
  });

  it("unlocks the repo via the menu and calls doRepoTask with UNLOCK", async () => {
    vi.mocked(backrestService.doRepoTask).mockResolvedValue({} as any);
    const successSpy = vi.spyOn(alerts, "success");

    const { user } = renderWithProviders(<PlanView plan={plan} />, { config });

    await openMenu(user);
    const unlockItem = await screen.findByText(m.repo_button_unlock());
    await user.click(unlockItem);

    await waitFor(() => {
      expect(backrestService.doRepoTask).toHaveBeenCalledWith(
        expect.objectContaining({
          repoId: "test-repo",
          task: DoRepoTaskRequest_Task.UNLOCK,
        }),
      );
    });
    expect(successSpy).toHaveBeenCalledWith(m.repo_success_unlocked());
  });

  it("clears error history via the menu with the plan/repo selector", async () => {
    vi.mocked(backrestService.clearHistory).mockResolvedValue({} as any);
    const successSpy = vi.spyOn(alerts, "success");

    const { user } = renderWithProviders(<PlanView plan={plan} />, { config });

    await openMenu(user);
    const clearHistoryItem = await screen.findByText(
      m.plan_button_clear_history(),
    );
    await user.click(clearHistoryItem);

    await waitFor(() => {
      expect(backrestService.clearHistory).toHaveBeenCalledWith(
        expect.objectContaining({
          selector: expect.objectContaining({
            planId: "test-plan",
            repoGuid: "test-repo-guid",
          }),
          onlyFailed: true,
        }),
      );
    });
    expect(successSpy).toHaveBeenCalledWith(m.plan_history_cleared());
  });
});
