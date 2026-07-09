import { screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import * as m from "../../paraglide/messages";
import {
  ScheduleDefaultsDaily,
  ScheduleDefaultsInfrequent,
  ScheduleFormItem,
} from "./ScheduleFormItem";
import { renderWithProviders } from "../../test/render";

// ScheduleFormItem is a controlled component operating on the protobuf-es
// toJson "flat" form-data shape (oneof fields hoisted to top-level keys, enum
// as its string NAME) rather than the nested { schedule: { case, value } }
// runtime shape - see the component's own comments. The harness below mirrors
// that contract: it holds the emitted value in state and re-renders, so mode
// switches and edits can be composed the same way a real form would use them.
const Harness = (props: {
  onChange: (val: any) => void;
  defaults?: typeof ScheduleDefaultsDaily;
  initial?: any;
}) => {
  const [value, setValue] = useState<any>(props.initial);
  return (
    <ScheduleFormItem
      value={value}
      defaults={props.defaults}
      onChange={(v) => {
        props.onChange(v);
        setValue(v);
      }}
    />
  );
};

describe("ScheduleFormItem", () => {
  it("initializes a missing clock to the defaults' clock enum name on mount", async () => {
    const onChange = vi.fn();
    renderWithProviders(
      <Harness onChange={onChange} defaults={ScheduleDefaultsDaily} />,
    );

    // ScheduleDefaultsDaily.clock = Schedule_Clock.LOCAL -> "CLOCK_LOCAL"
    expect(onChange).toHaveBeenCalledWith({ clock: "CLOCK_LOCAL" });
  });

  it("initializes clock to CLOCK_LAST_RUN_TIME under ScheduleDefaultsInfrequent", async () => {
    const onChange = vi.fn();
    renderWithProviders(
      <Harness onChange={onChange} defaults={ScheduleDefaultsInfrequent} />,
    );

    expect(onChange).toHaveBeenCalledWith({ clock: "CLOCK_LAST_RUN_TIME" });
  });

  it("switching to the days-interval mode applies defaults.maxFrequencyDays", async () => {
    const onChange = vi.fn();
    const { user } = renderWithProviders(
      <Harness onChange={onChange} defaults={ScheduleDefaultsInfrequent} />,
    );

    await user.click(
      screen.getByRole("button", {
        name: m.add_plan_modal_schedule_interval_days(),
      }),
    );

    expect(onChange).toHaveBeenLastCalledWith({
      clock: "CLOCK_LAST_RUN_TIME",
      maxFrequencyDays: ScheduleDefaultsInfrequent.maxFrequencyDays,
    });
    expect(ScheduleDefaultsInfrequent.maxFrequencyDays).toBe(30);
  });

  it("switching to the hours-interval mode applies defaults.maxFrequencyHours", async () => {
    const onChange = vi.fn();
    const { user } = renderWithProviders(
      <Harness onChange={onChange} defaults={ScheduleDefaultsDaily} />,
    );

    await user.click(
      screen.getByRole("button", {
        name: m.add_plan_modal_schedule_interval_hours(),
      }),
    );

    expect(onChange).toHaveBeenLastCalledWith({
      clock: "CLOCK_LOCAL",
      maxFrequencyHours: ScheduleDefaultsDaily.maxFrequencyHours,
    });
    expect(ScheduleDefaultsDaily.maxFrequencyHours).toBe(24);
  });

  it("switching to cron mode applies defaults.cron", async () => {
    const onChange = vi.fn();
    const { user } = renderWithProviders(
      <Harness onChange={onChange} defaults={ScheduleDefaultsDaily} />,
    );

    await user.click(
      screen.getByRole("button", { name: m.add_plan_modal_schedule_cron() }),
    );

    expect(onChange).toHaveBeenLastCalledWith({
      clock: "CLOCK_LOCAL",
      cron: ScheduleDefaultsDaily.cron,
    });
  });

  it("switching to disabled mode sets disabled: true and preserves clock", async () => {
    const onChange = vi.fn();
    const { user } = renderWithProviders(
      <Harness onChange={onChange} defaults={ScheduleDefaultsDaily} />,
    );

    await user.click(
      screen.getByRole("button", {
        name: m.add_plan_modal_schedule_disabled_label(),
      }),
    );

    expect(onChange).toHaveBeenLastCalledWith({
      clock: "CLOCK_LOCAL",
      disabled: true,
    });
    // Disabled mode hides the clock selector entirely.
    expect(
      screen.queryByText(m.add_plan_modal_schedule_reference_clock()),
    ).not.toBeInTheDocument();
  });

  it("editing the cron expression input propagates the new cron string", async () => {
    const onChange = vi.fn();
    const { user } = renderWithProviders(
      <Harness
        onChange={onChange}
        defaults={ScheduleDefaultsDaily}
        initial={{ clock: "CLOCK_LOCAL", cron: "0 0 * * *" }}
      />,
    );

    const cronInput = screen.getByDisplayValue("0 0 * * *");
    await user.type(cronInput, "1");

    expect(onChange).toHaveBeenLastCalledWith({
      clock: "CLOCK_LOCAL",
      cron: "0 0 * * *1",
    });
  });

  it("selecting a different reference clock updates the clock field only", async () => {
    const onChange = vi.fn();
    const { user } = renderWithProviders(
      <Harness
        onChange={onChange}
        defaults={ScheduleDefaultsDaily}
        initial={{ clock: "CLOCK_LOCAL", cron: "0 0 * * *" }}
      />,
    );

    await user.click(
      screen.getByRole("radio", { name: m.add_plan_modal_schedule_time_utc() }),
    );

    expect(onChange).toHaveBeenLastCalledWith({
      clock: "CLOCK_UTC",
      cron: "0 0 * * *",
    });
  });
});
