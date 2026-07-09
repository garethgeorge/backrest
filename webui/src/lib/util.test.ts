import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { debounce, groupBy, keyBy, namePattern } from "./util";

describe("debounce", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("invokes trailing call after the wait elapses", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, 100);

    debounced("a");
    expect(fn).not.toHaveBeenCalled();

    vi.advanceTimersByTime(99);
    expect(fn).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith("a");
  });

  it("coalesces repeated calls within the wait window into a single trailing invoke", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, 100);

    debounced("first");
    vi.advanceTimersByTime(50);
    debounced("second");
    vi.advanceTimersByTime(50);
    // Still within the wait window restarted by the second call.
    expect(fn).not.toHaveBeenCalled();

    vi.advanceTimersByTime(50);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith("second");
  });

  it("forces invocation once maxWait elapses even if calls keep resetting wait", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, 100, { maxWait: 250 });

    debounced("1");
    vi.advanceTimersByTime(80);
    debounced("2");
    vi.advanceTimersByTime(80);
    debounced("3");
    vi.advanceTimersByTime(80);
    // 240ms elapsed since the first call started maxWait timer; not yet at 250ms
    // and each call above kept resetting the 100ms trailing timer, so no trailing
    // fire should have happened yet.
    expect(fn).not.toHaveBeenCalled();

    debounced("4");
    vi.advanceTimersByTime(10);
    // maxWaitTimeoutId was scheduled at t=0 for 250ms; now at t=250 it should fire.
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith("4");
  });

  it("cancel() suppresses a pending trailing invocation", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, 100);

    debounced("a");
    debounced.cancel();

    vi.advanceTimersByTime(1000);
    expect(fn).not.toHaveBeenCalled();
  });
});

describe("groupBy", () => {
  it("groups items by the iteratee's key", () => {
    const items = [
      { type: "a", value: 1 },
      { type: "b", value: 2 },
      { type: "a", value: 3 },
    ];
    expect(groupBy(items, (i) => i.type)).toEqual({
      a: [
        { type: "a", value: 1 },
        { type: "a", value: 3 },
      ],
      b: [{ type: "b", value: 2 }],
    });
  });

  it("returns an empty object for an empty array", () => {
    expect(groupBy([] as { type: string }[], (i) => i.type)).toEqual({});
  });
});

describe("keyBy", () => {
  it("indexes items by the iteratee's key, last write wins on collision", () => {
    const items = [
      { id: "x", value: 1 },
      { id: "y", value: 2 },
      { id: "x", value: 3 },
    ];
    expect(keyBy(items, (i) => i.id)).toEqual({
      x: { id: "x", value: 3 },
      y: { id: "y", value: 2 },
    });
  });

  it("returns an empty object for an empty array", () => {
    expect(keyBy([] as { id: string }[], (i) => i.id)).toEqual({});
  });
});

describe("namePattern", () => {
  it.each(["a-b_c.1", "abc", "A1", "a.b-c_d", "123"])("accepts %s", (value) => {
    expect(namePattern.test(value)).toBe(true);
  });

  it.each(["a b", "a/b", "", "a@b", "a:b", "a#b"])("rejects %s", (value) => {
    expect(namePattern.test(value)).toBe(false);
  });
});
