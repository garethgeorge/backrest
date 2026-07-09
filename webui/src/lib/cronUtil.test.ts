import { describe, expect, it } from "vitest";
import { getMinimumCronDuration } from "./cronUtil";

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

describe("getMinimumCronDuration", () => {
  it("returns 0 for a single invocation regardless of schedule", () => {
    expect(getMinimumCronDuration("* * * * *", 1)).toBe(0);
    expect(getMinimumCronDuration("*/15 * * * *", 1)).toBe(0);
    expect(getMinimumCronDuration("0 * * * *", 1)).toBe(0);
    expect(getMinimumCronDuration("0 0 * * *", 1)).toBe(0);
    expect(getMinimumCronDuration("30 4 * * *", 1)).toBe(0);
    expect(getMinimumCronDuration("0 0 * * 1", 1)).toBe(0);
  });

  it("computes the span across evenly spaced minute invocations", () => {
    // Every minute: 5 invocations span 4 minutes.
    expect(getMinimumCronDuration("* * * * *", 5)).toBe(4 * MINUTE);
  });

  it("computes the span across a */15 minute schedule", () => {
    // 4 invocations of a 15-minute cadence span 3 intervals (45 minutes).
    expect(getMinimumCronDuration("*/15 * * * *", 4)).toBe(3 * 15 * MINUTE);
  });

  it("computes the span across an hourly schedule", () => {
    // Hourly: 3 invocations span 2 hours.
    expect(getMinimumCronDuration("0 * * * *", 3)).toBe(2 * HOUR);
  });

  it("computes the span across a daily schedule", () => {
    // Daily at midnight: 2 invocations span 1 day.
    expect(getMinimumCronDuration("0 0 * * *", 2)).toBe(1 * DAY);
  });

  it("computes the span across a weekly schedule", () => {
    // Weekly on Monday at midnight: 2 invocations span 7 days.
    expect(getMinimumCronDuration("0 0 * * 1", 2)).toBe(7 * DAY);
  });

  it("handles multiple minute values within an hour", () => {
    // Twice an hour (:00 and :30): 2 invocations span 30 minutes, 3 span 60.
    expect(getMinimumCronDuration("0,30 * * * *", 2)).toBe(30 * MINUTE);
    expect(getMinimumCronDuration("0,30 * * * *", 3)).toBe(60 * MINUTE);
  });

  it("handles multiple day-of-month values", () => {
    // Runs on the 1st and 15th at midnight: 2 invocations span 14 days.
    expect(getMinimumCronDuration("0 0 1,15 * *", 2)).toBe(14 * DAY);
  });

  it("defaults targetInvocations to 1 when omitted", () => {
    expect(getMinimumCronDuration("* * * * *")).toBe(0);
  });
});
