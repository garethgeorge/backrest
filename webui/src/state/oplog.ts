import {
  Operation,
  OperationEvent,
  OperationEventType,
  OperationStatus,
} from "../../gen/ts/v1/operations.pb";
import { GetOperationsRequest, Backrest } from "../../gen/ts/v1/service.pb";
import { API_PREFIX } from "../constants";
import { BackupProgressEntry, ResticSnapshot } from "../../gen/ts/v1/restic.pb";
import _ from "lodash";
import { formatDuration, formatTime } from "../lib/formatting";

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
      await Backrest.GetOperationEvents(
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
  const opList = await Backrest.GetOperations(req, {
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

export enum DisplayType {
  UNKNOWN,
  BACKUP,
  SNAPSHOT,
  FORGET,
  PRUNE,
  RESTORE,
}

export interface BackupInfo {
  id: string; // id of the first operation that generated this backup.
  displayTime: Date;
  displayType: DisplayType;
  startTimeMs: number;
  endTimeMs: number;
  status: OperationStatus;
  operations: Operation[]; // operations ordered by their unixTimeStartMs (not ID)
  repoId?: string;
  planId?: string;
  snapshotId?: string;
  backupLastStatus?: BackupProgressEntry;
  snapshotInfo?: ResticSnapshot;
  forgotten: boolean;
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

  private createBackup(operations: Operation[]): BackupInfo {
    // deduplicate and sort operations.
    operations.sort((a, b) => {
      return parseInt(b.unixTimeStartMs!) - parseInt(a.unixTimeStartMs!);
    });

    // use the lowest ID of all operations as the ID of the backup, this will be the first created operation.
    const id = operations.reduce((prev, curr) => {
      return prev < curr.id! ? prev : curr.id!;
    }, operations[0].id!);

    const startTimeMs = parseInt(operations[0].unixTimeStartMs!);
    const endTimeMs = parseInt(operations[operations.length - 1].unixTimeEndMs!);
    const displayTime = new Date(startTimeMs);
    let displayType = DisplayType.SNAPSHOT;
    if (operations.length === 1) {
      displayType = getTypeForDisplay(operations[0]);
    }

    // use the latest status that is not cancelled.
    let statusIdx = operations.length - 1;
    let status = OperationStatus.STATUS_SYSTEM_CANCELLED;
    while (statusIdx !== -1 && shouldHideStatus(status)) {
      status = operations[statusIdx].status!;
      statusIdx--;
    }

    let backupLastStatus = undefined;
    let snapshotInfo = undefined;
    let forgotten = false;
    for (const op of operations) {
      if (op.operationBackup) {
        backupLastStatus = op.operationBackup.lastStatus;
      } else if (op.operationIndexSnapshot) {
        snapshotInfo = op.operationIndexSnapshot.snapshot;
        forgotten = op.operationIndexSnapshot.forgot || false;
      }
    }

    return {
      id,
      startTimeMs,
      endTimeMs,
      displayTime,
      displayType,
      status,
      operations,
      backupLastStatus,
      snapshotInfo,
      forgotten,
      planId: operations[0].planId,
      repoId: operations[0].repoId,
    };
  }

  private addHelper(op: Operation): BackupInfo {
    if (op.snapshotId) {
      delete this.backupByOpId[op.id!];

      const existing = this.backupBySnapshotId[op.snapshotId!];
      let operations: Operation[];
      if (existing) {
        operations = [...existing.operations];
        const opIdx = operations.findIndex((o) => o.id === op.id);
        if (opIdx > -1) {
          operations[opIdx] = op;
        } else {
          operations.push(op);
        }
      } else {
        operations = [op];
      }

      const newInfo = this.createBackup(operations);
      this.backupBySnapshotId[op.snapshotId!] = newInfo;
      return newInfo;
    } else {
      const newInfo = this.createBackup([op]);
      this.backupByOpId[op.id!] = newInfo;
      return newInfo;
    }
  }

  public addOperation(event: OperationEventType, op: Operation): BackupInfo {
    const backupInfo = this.addHelper(op);
    this.listeners.forEach((l) => l(event, [backupInfo]));
    return backupInfo;
  }

  // removeOperaiton is not quite correct from a formal standpoint; but will look correct in the UI.
  public removeOperation(op: Operation) {
    if (op.snapshotId) {
      delete this.backupBySnapshotId[op.snapshotId];
    } else {
      delete this.backupByOpId[op.id!];
    }

    this.listeners.forEach((l) => l(OperationEventType.EVENT_DELETED, this.getAll()));
  }

  public bulkAddOperations(ops: Operation[]): BackupInfo[] {
    const backupInfos = ops.map((op) => this.addHelper(op));
    this.listeners.forEach((l) =>
      l(OperationEventType.EVENT_UNKNOWN, backupInfos)
    );
    return backupInfos;
  }

  public getAll(): BackupInfo[] {
    const arr = [];
    arr.push(...Object.values(this.backupByOpId));
    arr.push(...Object.values(this.backupBySnapshotId));
    return arr.filter((b) => !b.forgotten && !shouldHideStatus(b.status));
  }

  public subscribe(
    listener: (event: OperationEventType, info: BackupInfo[]) => void
  ) {
    this.listeners.push(listener);
  }

  public unsubscribe(
    listener: (event: OperationEventType, info: BackupInfo[]) => void
  ) {
    const index = this.listeners.indexOf(listener);
    if (index > -1) {
      this.listeners[index] = this.listeners[this.listeners.length - 1];
      this.listeners.pop();
    }
  }
}

export const shouldHideStatus = (status: OperationStatus) => {
  return status === OperationStatus.STATUS_SYSTEM_CANCELLED;
};

export const getTypeForDisplay = (op: Operation) => {
  if (op.operationForget) {
    return DisplayType.FORGET;
  } else if (op.operationPrune) {
    return DisplayType.PRUNE;
  } else if (op.operationBackup) {
    return DisplayType.BACKUP;
  } else if (op.operationIndexSnapshot) {
    return DisplayType.SNAPSHOT;
  } else if (op.operationPrune) {
    return DisplayType.PRUNE;
  } else if (op.operationRestore) {
    return DisplayType.RESTORE;
  } else {
    return DisplayType.UNKNOWN;
  }
};

export const displayTypeToString = (type: DisplayType) => {
  switch (type) {
    case DisplayType.BACKUP:
      return "Backup";
    case DisplayType.SNAPSHOT:
      return "Snapshot";
    case DisplayType.FORGET:
      return "Forget";
    case DisplayType.PRUNE:
      return "Prune";
    case DisplayType.RESTORE:
      return "Restore";
    default:
      return "Unknown";
  }
};
// detailsForOperation returns derived display information for a given operation.
export const detailsForOperation = (
  op: Operation
): {
  state: string;
  displayState: string;
  duration: number;
  percentage?: number;
  color: string;
} => {
  let state = "";
  let duration = 0;
  let percentage: undefined | number = undefined;
  let color: string;
  switch (op.status!) {
    case OperationStatus.STATUS_PENDING:
      state = "pending";
      color = "grey";
      break;
    case OperationStatus.STATUS_INPROGRESS:
      state = "runnning";
      duration = new Date().getTime() - parseInt(op.unixTimeStartMs!);
      color = "blue";
      break;
    case OperationStatus.STATUS_ERROR:
      state = "error";
      color = "red";
      break;
    case OperationStatus.STATUS_WARNING:
      state = "warning";
      color = "orange";
      break;
    case OperationStatus.STATUS_SUCCESS:
      state = "";
      color = "green";
      break;
    case OperationStatus.STATUS_USER_CANCELLED:
      state = "cancelled";
      color = "orange";
      break;
    default:
      state = "";
      color = "grey";
  }

  switch (op.status) {
    case OperationStatus.STATUS_INPROGRESS:
      duration = new Date().getTime() - parseInt(op.unixTimeStartMs!);

      if (op.operationBackup) {
        const backup = op.operationBackup;
        if (backup.lastStatus && backup.lastStatus.status) {
          percentage = (backup.lastStatus.status.percentDone || 0) * 100;
        } else if (backup.lastStatus && backup.lastStatus.summary) {
          percentage = 100;
        }
      } else if (op.operationRestore) {
        const restore = op.operationRestore;
        if (restore.status) {
          percentage = (restore.status.percentDone || 1) * 100;
        }
      }

      break;
    default:
      duration = parseInt(op.unixTimeEndMs!) - parseInt(op.unixTimeStartMs!);
      break;
  }

  let displayState = state;
  if (duration > 0) {
    if (op.status === OperationStatus.STATUS_INPROGRESS) {
      displayState += " for " + formatDuration(duration);
    } else {
      displayState += " in " + formatDuration(duration);
    }
  }

  if (percentage !== undefined) {
    displayState += ` (${percentage.toFixed(2)}%)`;
  }

  return {
    state,
    displayState,
    duration,
    percentage,
    color,
  };
};
