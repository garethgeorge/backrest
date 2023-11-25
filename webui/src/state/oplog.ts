import {
  Operation,
  OperationEvent,
  OperationEventType,
  OperationStatus,
} from "../../gen/ts/v1/operations.pb";
import { GetOperationsRequest, ResticUI } from "../../gen/ts/v1/service.pb";
import { API_PREFIX } from "../constants";
import { BackupProgressEntry, ResticSnapshot } from "../../gen/ts/v1/restic.pb";
import _ from "lodash";

export type EOperation = Operation & {
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

export const getOperations = async (
  req: GetOperationsRequest
): Promise<EOperation[]> => {
  const opList = await ResticUI.GetOperations(req, {
    pathPrefix: API_PREFIX,
  });
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
      (o) => !operations.find((op) => op.id === o.id)
    );
    operations.sort((a, b) => {
      return a.parsedTime! - b.parsedTime!;
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
          return a.parsedTime! - b.parsedTime!;
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
    parsedTime: time,
    parsedDate: date,
  };
};

export interface BackupInfo {
  id: string; // id of the first operation that generated this backup.
  displayTime: Date;
  startTimeMs: number;
  endTimeMs: number;
  status: OperationStatus;
  operations: Operation[];
  repoId?: string;
  planId?: string;
  snapshotId?: string;
  backupLastStatus?: BackupProgressEntry;
  snapshotInfo?: ResticSnapshot;
}

// BackupInfoCollector maps multiple operations to single aggregate 'BackupInfo' objects.
// A backup info object aggregates the backup status (if available), snapshot info (if available), and possibly the forget status (if available).
export class BackupInfoCollector {
  private listeners: ((
    event: OperationEventType,
    info: BackupInfo[]
  ) => void)[] = [];
  private backupByOpId: { [key: string]: BackupInfo } = {};
  private backupBySnapshotId: { [key: string]: BackupInfo } = {};

  private mergeBackups(existing: BackupInfo, newInfo: BackupInfo) {
    existing.startTimeMs = Math.min(existing.startTimeMs, newInfo.startTimeMs);
    existing.endTimeMs = Math.max(existing.endTimeMs, newInfo.endTimeMs);
    existing.displayTime = new Date(existing.startTimeMs);
    if (newInfo.startTimeMs >= existing.startTimeMs) {
      existing.status = newInfo.status; // use the latest status
    }
    existing.operations = _.uniqBy(
      [...existing.operations, ...newInfo.operations],
      (o) => o.id!
    );
    if (newInfo.backupLastStatus) {
      existing.backupLastStatus = newInfo.backupLastStatus;
    }
    if (newInfo.snapshotInfo) {
      existing.snapshotInfo = newInfo.snapshotInfo;
    }
    return existing;
  }

  private operationToBackup(op: Operation): BackupInfo {
    const startTimeMs = parseInt(op.unixTimeStartMs!);
    const endTimeMs = parseInt(op.unixTimeEndMs!);

    const b: BackupInfo = {
      id: op.id!,
      startTimeMs,
      endTimeMs,
      status: op.status!,
      displayTime: new Date(startTimeMs),
      repoId: op.repoId,
      planId: op.planId,
      snapshotId: op.snapshotId,
      operations: [op],
    };

    if (op.operationBackup) {
      const ob = op.operationBackup;
      b.backupLastStatus = ob.lastStatus;
    }
    if (op.operationIndexSnapshot) {
      const oi = op.operationIndexSnapshot;
      b.snapshotInfo = oi.snapshot;
    }
    return b;
  }

  private addHelper(op: Operation) {
    if (op.snapshotId) {
      delete this.backupByOpId[op.id!];

      let newInfo = this.operationToBackup(op);
      const existing = this.backupBySnapshotId[op.snapshotId!];
      if (existing) {
        this.mergeBackups(existing, newInfo);
        return existing;
      } else {
        this.backupBySnapshotId[op.snapshotId] = newInfo;
        return newInfo;
      }
    } else {
      const newInfo = this.operationToBackup(op);
      this.backupByOpId[op.id!] = newInfo;
      return newInfo;
    }
  }

  public addOperation(event: OperationEventType, op: Operation): BackupInfo {
    const backupInfo = this.addHelper(op);
    this.listeners.forEach((l) => l(event, [backupInfo]));
    return backupInfo;
  }

  public bulkAddOperations(ops: Operation[]): BackupInfo[] {
    const backupInfos = ops.map((op) => this.addHelper(op));
    this.listeners.forEach((l) =>
      l(OperationEventType.EVENT_UNKNOWN, backupInfos)
    );
    return backupInfos;
  }

  public addOperationNoNotify(op: Operation) {
    this.addHelper(op);
  }

  public getAll(): BackupInfo[] {
    const arr = [];
    arr.push(...Object.values(this.backupByOpId));
    arr.push(...Object.values(this.backupBySnapshotId));
    return arr;
  }

  public subscribe(
    listener: (event: OperationEventType, info: BackupInfo) => void
  ) {
    this.listeners.push(listener);
  }

  public unsubscribe(
    listener: (event: OperationEventType, info: BackupInfo) => void
  ) {
    const index = this.listeners.indexOf(listener);
    if (index > -1) {
      this.listeners[index] = this.listeners[this.listeners.length - 1];
      this.listeners.pop();
    }
  }
}
