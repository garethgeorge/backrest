import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import * as m from "../../paraglide/messages";
import { DynamicList } from "./DynamicList";
import { renderWithProviders } from "../../test/render";

// NOTE: drag-to-reorder (dnd-kit's PointerSensor + DragEndEvent) relies on
// browser pointer-event geometry that jsdom does not implement, so reordering
// via drag is intentionally not exercised here. The add/remove/edit paths
// below cover the same onUpdate contract without needing real drag gestures.

describe("DynamicList", () => {
  it("renders one input per item plus an Add control", () => {
    const onUpdate = vi.fn();
    renderWithProviders(
      <DynamicList label="Paths" items={["a", "b"]} onUpdate={onUpdate} />,
    );

    const inputs = screen.getAllByRole("textbox");
    expect(inputs).toHaveLength(2);
    expect(inputs[0]).toHaveValue("a");
    expect(inputs[1]).toHaveValue("b");
    expect(
      screen.getByRole("button", { name: m.add_plan_modal_field_add() }),
    ).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Remove" })).toHaveLength(2);
  });

  it("appends an empty item and calls onUpdate when the Add control is clicked", async () => {
    const onUpdate = vi.fn();
    const { user } = renderWithProviders(
      <DynamicList label="Paths" items={["a", "b"]} onUpdate={onUpdate} />,
    );

    await user.click(
      screen.getByRole("button", { name: m.add_plan_modal_field_add() }),
    );

    expect(onUpdate).toHaveBeenCalledTimes(1);
    expect(onUpdate).toHaveBeenCalledWith(["a", "b", ""]);
  });

  it("removes the targeted item and calls onUpdate with the item excluded", async () => {
    const onUpdate = vi.fn();
    const { user } = renderWithProviders(
      <DynamicList label="Paths" items={["a", "b", "c"]} onUpdate={onUpdate} />,
    );

    const removeButtons = screen.getAllByRole("button", { name: "Remove" });
    await user.click(removeButtons[1]); // remove "b"

    expect(onUpdate).toHaveBeenCalledTimes(1);
    expect(onUpdate).toHaveBeenCalledWith(["a", "c"]);
  });

  it("propagates edits typed into an item's input through onUpdate", async () => {
    const onUpdate = vi.fn();
    const { user } = renderWithProviders(
      <DynamicList label="Paths" items={["a", "b"]} onUpdate={onUpdate} />,
    );

    const inputs = screen.getAllByRole("textbox");
    await user.type(inputs[0], "x");

    // Each keystroke fires onChange against the original `items` prop (it does
    // not update local state), so the last call reflects "a" + "x" typed once.
    expect(onUpdate).toHaveBeenCalledWith(["ax", "b"]);
  });
});
