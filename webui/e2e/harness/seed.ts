import { createClient, type Client } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { create } from '@bufbuild/protobuf';
import { AddRepoRequestSchema, Backrest } from '../../gen/ts/v1/service_pb';
import {
  AuthSchema,
  PlanSchema,
  Schedule_Clock,
  type Config,
  type Plan,
  type Repo,
} from '../../gen/ts/v1/config_pb';
import type { BackrestInstance } from './backrest';

/**
 * ConnectRPC seeding helpers. These talk directly to the backrest API (no
 * browser involved) so tests can set up state quickly and use the UI only for
 * the behavior under test.
 *
 * SetConfig uses optimistic concurrency via Config.modno: always GetConfig
 * first, mutate the returned message, and send it back (the server bumps
 * modno). These helpers follow that pattern; avoid calling them concurrently
 * against the same instance.
 */

export function backrestClient(inst: BackrestInstance): Client<typeof Backrest> {
  return createClient(Backrest, createConnectTransport({ baseUrl: inst.url }));
}

/**
 * First-run setup: names the instance and disables authentication so the UI
 * loads without the initial-setup Settings modal or a login gate.
 */
export async function seedInstance(inst: BackrestInstance, name = 'e2e-test'): Promise<Config> {
  const client = backrestClient(inst);
  const config = await client.getConfig({});
  config.instance = name;
  config.auth = create(AuthSchema, { disabled: true });
  return await client.setConfig(config);
}

/**
 * Adds (and initializes) a local restic repository at inst.repoPath(id).
 * Returns the repo as recorded in the resulting config.
 */
export async function seedRepo(
  inst: BackrestInstance,
  id = 'local-repo',
  password = 'test-password-12345',
): Promise<Repo> {
  const client = backrestClient(inst);
  const config = await client.addRepo(
    create(AddRepoRequestSchema, {
      repo: {
        id,
        uri: inst.repoPath(id),
        password,
      },
    }),
  );
  const repo = config.repos.find((r) => r.id === id);
  if (!repo) {
    throw new Error(`seedRepo: repo ${id} missing from config after AddRepo`);
  }
  return repo;
}

/**
 * Appends a plan to the config. The schedule is explicitly disabled so no
 * background backups fire during the test; trigger backups via the UI
 * ("Backup Now") or the Backup RPC.
 */
export async function seedPlan(
  inst: BackrestInstance,
  id: string,
  repoId: string,
  paths: string[],
): Promise<Plan> {
  const client = backrestClient(inst);
  const config = await client.getConfig({});
  const plan = create(PlanSchema, {
    id,
    repo: repoId,
    paths,
    schedule: {
      schedule: { case: 'disabled', value: true },
      clock: Schedule_Clock.LOCAL,
    },
  });
  config.plans.push(plan);
  await client.setConfig(config);
  return plan;
}
