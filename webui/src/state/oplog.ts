import { atom } from "recoil";
import {
  Operation,
  OperationEvent,
  OperationEventType,
  OperationStatus,
} from "../../gen/ts/v1/operations.pb";
import { GetOperationsRequest, ResticUI } from "../../gen/ts/v1/service.pb";
import { EventEmitter } from "events";
import { useAlertApi } from "../components/Alerts";
import { API_PREFIX } from "../constants";
import { BackupProgressEntry, ResticSnapshot } from "../../gen/ts/v1/restic.pb";

export type EOperation = Operation & {
  parsedId: number;
  parsedTime: number;
  parsedDate: Date;
};

const subscribers: ((event: OperationEvent) => void)[] = [];

// Start fetching and emitting operations.
(async () => {
  while (true) {
    let nextConnWaitUntil = new Date().getTime() + 5000;
    try {
      await ResticUI.GetOperationEvents(
        {},
        (event: OperationEvent) => {
          console.log("operation event", event);
          subscribers.forEach((subscriber) => subscriber(event));
        },
        {
          pathPrefix: API_PREFIX,
        }
      );
    } catch (e: any) {
      console.error("operations stream died with exception: ", e);
    }
    await new Promise((accept, _) =>
      setTimeout(accept, nextConnWaitUntil - new Date().getTime())
    );
  }
})();

export const getOperations = async ({
  planId,
  repoId,
  lastN,
}: GetOperationsRequest): Promise<EOperation[]> => {
  const opList = await ResticUI.GetOperations(
    {
      planId,
      repoId,
      lastN,
    },
    {
      pathPrefix: API_PREFIX,
    }
  );
  return (opList.operations || []).map(toEop);
};

export const subscribeToOperations = (
  callback: (event: OperationEvent) => void
) => {
  subscribers.push(callback);
};

export const unsubscribeFromOperations = (
  callback: (event: OperationEvent) => void
) => {
  const index = subscribers.indexOf(callback);
  if (index > -1) {
    subscribers[index] = subscribers[subscribers.length - 1];
    subscribers.pop();
  }
};

export const buildOperationListListener = (
  req: GetOperationsRequest,
  callback: (
    event: OperationEventType | null,
    operation: EOperation | null,
    list: EOperation[]
  ) => void
) => {
  let operations: EOperation[] = [];

  (async () => {
    let opsFromServer = await getOperations(req);
    operations = opsFromServer.filter(
      (o) => !operations.find((op) => op.parsedId === o.parsedId)
    );
    operations.sort((a, b) => {
      return a.parsedId - b.parsedId;
    });

    callback(null, null, operations);
  })();

  return (event: OperationEvent) => {
    const op = toEop(event.operation!);
    const type = event.type!;
    if (!!req.planId && op.planId !== req.planId) {
      return;
    }
    if (!!req.repoId && op.repoId !== req.repoId) {
      return;
    }
    if (type === OperationEventType.EVENT_UPDATED) {
      const index = operations.findIndex((o) => o.id === op.id);
      if (index > -1) {
        operations[index] = op;
      } else {
        operations.push(op);
        operations.sort((a, b) => {
          return a.parsedId - b.parsedId;
        });
      }
    } else if (type === OperationEventType.EVENT_CREATED) {
      operations.push(op);
    }

    callback(event.type || null, op, operations);
  };
};

export const toEop = (op: Operation): EOperation => {
  const time = parseInt(op.unixTimeStartMs!);
  const date = new Date();
  date.setTime(time);

  return {
    ...op,
    parsedId: parseInt(op.id!),
    parsedTime: time,
    parsedDate: date,
  };
};

// TODO: aggregate backup info from oplog.
interface BackupInfo {
  opids: string[];
  repoId: string;
  planId: string;
  backupLastStatus?: BackupProgressEntry;
  snapshotInfo?: ResticSnapshot;
}
