import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  ConfigSchema,
  AuthSchema,
  UserSchema,
} from "../../gen/ts/v1/config_pb";
import { makeConfig, makeFirstRunConfig } from "../test/proto";
import { shouldShowSettings } from "./configutil";

describe("shouldShowSettings", () => {
  it("is true when there is no instance configured (first run)", () => {
    expect(shouldShowSettings(makeFirstRunConfig())).toBe(true);
  });

  it("is true when an instance is configured but auth is unset", () => {
    const config = create(ConfigSchema, {
      instance: "test-instance",
    });
    expect(config.auth).toBeUndefined();
    expect(shouldShowSettings(config)).toBe(true);
  });

  it("is true when auth is enabled (not disabled) but has zero users", () => {
    const config = makeConfig({
      auth: create(AuthSchema, { disabled: false, users: [] }),
    });
    expect(shouldShowSettings(config)).toBe(true);
  });

  it("is false when instance is set and auth is explicitly disabled", () => {
    const config = makeConfig({
      auth: create(AuthSchema, { disabled: true, users: [] }),
    });
    expect(shouldShowSettings(config)).toBe(false);
  });

  it("is false when instance is set, auth is enabled, and at least one user exists", () => {
    const config = makeConfig({
      auth: create(AuthSchema, {
        disabled: false,
        users: [create(UserSchema, { name: "admin" })],
      }),
    });
    expect(shouldShowSettings(config)).toBe(false);
  });
});
