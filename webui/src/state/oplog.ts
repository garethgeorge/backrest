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
    existing.displayType = DisplayType.SNAPSHOT;
    if (newInfo.startTimeMs >= existing.startTimeMs) {
      existing.status = newInfo.status; // use the latest status
    }
    existing.operations = _.uniqBy(
      [...newInfo.operations, ...existing.operations],
      (o) => o.id!
    );
    existing.operations.sort((a, b) => {
      return parseInt(a.unixTimeStartMs!) - parseInt(b.unixTimeStartMs!);
    });
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
      displayType: getTypeForDisplay(op),
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
    arr.push(
      ...Object.values(this.backupBySnapshotId).filter((b) => {
        for (const op of b.operations) {
          if (op.operationIndexSnapshot && op.operationIndexSnapshot.forgot) {
            return false;
          }
        }
        return true;
      })
    );
    return arr;
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
