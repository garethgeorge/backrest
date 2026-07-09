import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import * as m from "../../paraglide/messages";
import { RetentionPolicyView } from "./RetentionPolicyView";
import { renderWithProviders } from "../../test/render";

describe("RetentionPolicyView", () => {
  it("defaults to the time-bucketed mode when retention is undefined", () => {
    renderWithProviders(
      <RetentionPolicyView retention={undefined} onChange={vi.fn()} />,
    );

    expect(
      screen.getByRole("button", {
        name: m.add_plan_modal_retention_policy_mode_time_label(),
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(m.add_plan_modal_retention_policy_hourly_label()),
    ).toBeInTheDocument();
    expect(
      screen.getByText(m.add_plan_modal_retention_policy_yearly_label()),
    ).toBeInTheDocument();
  });

  it("switches to keep-last-N mode and reports policyKeepLastN: 30", async () => {
    const onChange = vi.fn();
    const { user } = renderWithProviders(
      <RetentionPolicyView retention={undefined} onChange={onChange} />,
    );

    await user.click(
      screen.getByRole("button", {
        name: m.add_plan_modal_retention_policy_mode_count_label(),
      }),
    );

    expect(onChange).toHaveBeenCalledWith({ policyKeepLastN: 30 });
  });

  it("switches to keep-all (none) mode and reports policyKeepAll: true", async () => {
    const onChange = vi.fn();
    const { user } = renderWithProviders(
      <RetentionPolicyView retention={undefined} onChange={onChange} />,
    );

    await user.click(
      screen.getByRole("button", {
        name: m.add_plan_modal_retention_policy_mode_none_label(),
      }),
    );

    expect(onChange).toHaveBeenCalledWith({ policyKeepAll: true });
  });

  it("renders the keep-all warning when already in policyKeepAll mode", () => {
    renderWithProviders(
      <RetentionPolicyView
        retention={{ policyKeepAll: true }}
        onChange={vi.fn()}
      />,
    );

    expect(
      screen.getByText(m.add_plan_modal_retention_policy_keep_all_warning()),
    ).toBeInTheDocument();
  });

  it("renders the keep-last-N count in policyKeepLastN mode", () => {
    renderWithProviders(
      <RetentionPolicyView
        retention={{ policyKeepLastN: 30 }}
        onChange={vi.fn()}
      />,
    );

    expect(
      screen.getByRole("spinbutton", {
        name: m.add_plan_modal_retention_policy_keep_last_n_snapshots_label(),
      }),
    ).toHaveValue("30");
  });

  it("edits a time-bucketed field and preserves sibling fields via updateRetentionField", async () => {
    const onChange = vi.fn();
    const retention = {
      policyTimeBucketed: {
        yearly: 1,
        monthly: 3,
        weekly: 4,
        daily: 7,
        hourly: 24,
        keepLastN: 0,
      },
    };
    const { user } = renderWithProviders(
      <RetentionPolicyView retention={retention} onChange={onChange} />,
    );

    const dailyInput = screen.getByRole("spinbutton", {
      name: m.add_plan_modal_retention_policy_daily_label(),
    });
    await user.clear(dailyInput);
    await user.type(dailyInput, "9");

    expect(onChange).toHaveBeenLastCalledWith({
      policyTimeBucketed: {
        yearly: 1,
        monthly: 3,
        weekly: 4,
        daily: 9,
        hourly: 24,
        keepLastN: 0,
      },
    });
  });

  it("shows the high-frequency helper text when the schedule is a sub-hourly cron", () => {
    const retention = {
      policyTimeBucketed: {
        yearly: 0,
        monthly: 0,
        weekly: 0,
        daily: 0,
        hourly: 0,
        keepLastN: 5,
      },
    };
    const subHourlySchedule = {
      schedule: { case: "cron", value: "*/5 * * * *" },
    };

    renderWithProviders(
      <RetentionPolicyView
        schedule={subHourlySchedule}
        retention={retention}
        onChange={vi.fn()}
      />,
    );

    expect(
      screen.getByText(
        "Keep recent snapshots (High-frequency schedule detected)",
      ),
    ).toBeInTheDocument();
  });

  it("policyKeepLastN edits preserve sibling retention fields and never leak schedule fields", async () => {
    // Regression test: this branch used to spread the `schedule` prop instead
    // of `retention`, dropping sibling retention fields and leaking schedule
    // fields into the retention payload.
    const onChange = vi.fn();
    const scheduleProp = { schedule: { case: "cron", value: "0 0 * * *" } };
    const { user } = renderWithProviders(
      <RetentionPolicyView
        schedule={scheduleProp}
        retention={{ policyKeepLastN: 30 }}
        onChange={onChange}
      />,
    );

    const input = screen.getByRole("spinbutton", {
      name: m.add_plan_modal_retention_policy_keep_last_n_snapshots_label(),
    });
    await user.clear(input);
    await user.type(input, "5");

    expect(onChange).toHaveBeenLastCalledWith({ policyKeepLastN: 5 });
  });
});
