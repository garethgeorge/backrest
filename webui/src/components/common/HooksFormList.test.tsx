import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import * as m from "../../paraglide/messages";
import { HooksFormList, HookFields } from "./HooksFormList";
import { renderWithProviders } from "../../test/render";

// HooksFormList manages its own list state via Chakra's useControllableState:
// when no `value` prop is passed it behaves uncontrolled (seeded from
// `defaultValue`), so onChange calls also reflow into what's on screen without
// needing a controlled harness wrapper (unlike DynamicList's onUpdate, which
// is not wired back to a re-render).

const openAddHookMenu = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.click(
    screen.getByRole("button", { name: m.add_plan_modal_field_add_hook() }),
  );
};

describe("HooksFormList", () => {
  it("renders an empty state with just the add-hook control", () => {
    const onChange = vi.fn();
    renderWithProviders(<HooksFormList onChange={onChange} />);

    expect(
      screen.getByRole("button", { name: m.add_plan_modal_field_add_hook() }),
    ).toBeInTheDocument();
    expect(screen.queryByText(/Hook 1:/)).not.toBeInTheDocument();
  });

  it("adding a COMMAND hook appends an entry and propagates through onChange", async () => {
    const onChange = vi.fn();
    const { user } = renderWithProviders(<HooksFormList onChange={onChange} />);

    await openAddHookMenu(user);
    await user.click(
      await screen.findByRole("menuitem", {
        name: m.repo_hooks_command_label(),
      }),
    );

    expect(onChange).toHaveBeenCalledTimes(1);
    const emitted = onChange.mock.calls[0][0] as HookFields[];
    expect(emitted).toHaveLength(1);
    expect(emitted[0]).toEqual({
      conditions: [],
      actionCommand: { command: "echo {{ .ShellEscape .Summary }}" },
    });

    // Re-rendered card reflects the new hook.
    expect(
      screen.getByText(`Hook 1: ${m.repo_hooks_command_label()}`),
    ).toBeInTheDocument();
  });

  it("shows the command textarea for a COMMAND hook and propagates typed text", async () => {
    const onChange = vi.fn();
    const initial: HookFields[] = [
      { conditions: [], actionCommand: { command: "" } },
    ];
    const { user } = renderWithProviders(
      <HooksFormList defaultValue={initial} onChange={onChange} />,
    );

    // The command textarea is the only <textarea> rendered; conditions and
    // on-error selectors render as buttons, not textboxes.
    const textboxes = screen.getAllByRole("textbox");
    expect(textboxes).toHaveLength(1);

    await user.type(textboxes[0], "echo hi");

    expect(onChange).toHaveBeenLastCalledWith([
      { conditions: [], actionCommand: { command: "echo hi" } },
    ]);
  });

  it("selecting a condition propagates the enum value on the target hook", async () => {
    const onChange = vi.fn();
    const initial: HookFields[] = [
      { conditions: [], actionCommand: { command: "" } },
    ];
    renderWithProviders(
      <HooksFormList defaultValue={initial} onChange={onChange} />,
    );
    // Chakra's Select trigger relies on zag.js pointer-capture logic that the
    // default userEvent pointer-events check rejects in jsdom; disable it for
    // this interaction (mirrors AddPlanModal.test.tsx's newUser()). The trigger
    // gets a real accessible name (auto-derived from its placeholder by
    // SelectTrigger), so locate it by role/name.
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    const conditionsTrigger = screen.getByRole("combobox", {
      name: m.repo_hooks_command_runs_when(),
    });
    await user.click(conditionsTrigger);

    // Each option's accessible name is its enum label plus description text
    // concatenated ("CONDITION_SNAPSHOT_SUCCESS- Triggered when..."), so
    // match on a name prefix rather than an exact string.
    await user.click(
      await screen.findByRole("option", {
        name: /^CONDITION_SNAPSHOT_SUCCESS/,
      }),
    );

    expect(onChange).toHaveBeenLastCalledWith([
      {
        conditions: ["CONDITION_SNAPSHOT_SUCCESS"],
        actionCommand: { command: "" },
      },
    ]);
  });

  it("removes the targeted hook and leaves the other entries intact", async () => {
    const onChange = vi.fn();
    const initial: HookFields[] = [
      {
        conditions: ["CONDITION_SNAPSHOT_SUCCESS"],
        actionCommand: { command: "a" },
      },
      { conditions: [], actionCommand: { command: "b" } },
    ];
    const { user } = renderWithProviders(
      <HooksFormList defaultValue={initial} onChange={onChange} />,
    );

    expect(screen.getAllByText(/^Hook \d+:/)).toHaveLength(2);

    const removeButtons = screen.getAllByRole("button", {
      name: "Remove hook",
    });
    await user.click(removeButtons[0]);

    expect(onChange).toHaveBeenCalledWith([
      { conditions: [], actionCommand: { command: "b" } },
    ]);
    expect(screen.getAllByText(/^Hook \d+:/)).toHaveLength(1);
  });

  it("adding a Discord hook shows its webhook field and propagates edits", async () => {
    const onChange = vi.fn();
    const { user } = renderWithProviders(<HooksFormList onChange={onChange} />);

    await openAddHookMenu(user);
    await user.click(await screen.findByRole("menuitem", { name: "Discord" }));

    expect(onChange).toHaveBeenLastCalledWith([
      {
        conditions: [],
        actionDiscord: { webhookUrl: "", template: "{{ .Summary }}" },
      },
    ]);

    const urlInput = screen.getByPlaceholderText("Discord Webhook URL");
    await user.type(urlInput, "https://example.com/hook");

    expect(onChange).toHaveBeenLastCalledWith([
      {
        conditions: [],
        actionDiscord: {
          webhookUrl: "https://example.com/hook",
          template: "{{ .Summary }}",
        },
      },
    ]);
  });
});
