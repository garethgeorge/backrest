import { create, MessageInitShape } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import {
  Config,
  ConfigSchema,
  Plan,
  PlanSchema,
  Repo,
  RepoSchema,
} from "../../gen/ts/v1/config_pb";

/** A configured instance: shouldShowSettings(makeConfig()) === false. */
export const makeConfig = (
  overrides?: MessageInitShape<typeof ConfigSchema>,
): Config =>
  create(ConfigSchema, {
    instance: "test-instance",
    auth: { disabled: true, users: [] },
    repos: [],
    plans: [],
    ...overrides,
  } as MessageInitShape<typeof ConfigSchema>);

/** A virgin config, as returned by a freshly started backend. */
export const makeFirstRunConfig = (): Config => create(ConfigSchema, {});

export const makeRepo = (
  overrides?: MessageInitShape<typeof RepoSchema>,
): Repo =>
  create(RepoSchema, {
    id: "test-repo",
    guid: "test-repo-guid",
    uri: "/tmp/test-repo",
    password: "test-password",
    ...overrides,
  } as MessageInitShape<typeof RepoSchema>);

export const makePlan = (
  overrides?: MessageInitShape<typeof PlanSchema>,
): Plan =>
  create(PlanSchema, {
    id: "test-plan",
    repo: "test-repo",
    paths: ["/tmp/data-to-backup"],
    ...overrides,
  } as MessageInitShape<typeof PlanSchema>);

/**
 * Builds a ConnectError for mock rejections. AuthenticationBoundary branches
 * on Code.Unauthenticated / Unavailable / DeadlineExceeded.
 */
export const connectError = (code: Code, message = "test error") =>
  new ConnectError(message, code);

export { Code };
