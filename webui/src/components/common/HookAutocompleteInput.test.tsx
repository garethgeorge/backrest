import React from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { HookAutocompleteInput } from "./HookAutocompleteInput";
import { renderWithProviders } from "../../test/render";
import { backrestService } from "../../test/mocks/client";
import { create } from "@bufbuild/protobuf";
import { StringListSchema } from "../../../gen/ts/types/value_pb";

describe("HookAutocompleteInput", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    backrestService.hookAutocomplete.mockResolvedValue(
      create(StringListSchema, { values: ["abc", "def", "ghi"] })
    );
  });

  it("allows typing a custom value without losing focus (parent re-renders on each keystroke)", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });

    // Parent simulates the real form: state lives in parent, each keystroke
    // triggers setFormData -> re-render -> value prop changes.
    const Wrapper = () => {
      const [value, setValue] = React.useState("");
      return (
        <HookAutocompleteInput
          value={value}
          onChange={setValue}
          action="actionTelegram"
          field="botToken"
          placeholder="Bot Token"
        />
      );
    };

    renderWithProviders(<Wrapper />);

    const input = screen.getByPlaceholderText("Bot Token");
    await user.click(input);
    await user.type(input, "my-custom-token");

    expect(input).toHaveValue("my-custom-token");
  });

  it("shows dropdown on focus without typing", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });

    const Wrapper = () => {
      const [value, setValue] = React.useState("");
      return (
        <HookAutocompleteInput
          value={value}
          onChange={setValue}
          action="actionTelegram"
          field="botToken"
          placeholder="Bot Token"
        />
      );
    };

    renderWithProviders(<Wrapper />);

    const input = screen.getByPlaceholderText("Bot Token");
    await user.click(input);

    // Dropdown should appear with fetched suggestions
    const option = await screen.findByText("abc");
    expect(option).toBeInTheDocument();
    expect(screen.getByText("def")).toBeInTheDocument();
    expect(screen.getByText("ghi")).toBeInTheDocument();
  });

  it("selecting a suggestion fills the input", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });

    const Wrapper = () => {
      const [value, setValue] = React.useState("");
      return (
        <HookAutocompleteInput
          value={value}
          onChange={setValue}
          action="actionTelegram"
          field="botToken"
          placeholder="Bot Token"
        />
      );
    };

    renderWithProviders(<Wrapper />);

    const input = screen.getByPlaceholderText("Bot Token");
    await user.click(input);

    const option = await screen.findByText("abc");
    await user.click(option);

    expect(input).toHaveValue("abc");
  });
});
