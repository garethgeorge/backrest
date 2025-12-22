import { Operation, OperationStatus } from "../../gen/ts/v1/operations_pb";
import { formatBytes, formatDuration, normalizeSnapshotId } from "../lib/formatting";
import * as m from "../paraglide/messages";

export enum DisplayType {
  UNKNOWN,
  BACKUP,
  SNAPSHOT,
  FORGET,
  PRUNE,
  CHECK,
  RESTORE,
  STATS,
  RUNHOOK,
  RUNCOMMAND,
}

export interface FlowDisplayInfo {
  displayTime: number,
  flowID: bigint,
  planID: string,
  repoID: string,
  instanceID: string,
  snapshotID: string,
  status: OperationStatus,
  type: DisplayType;
  subtitleComponents: string[];
  hidden: boolean;
  operations: Operation[];
}

export const displayInfoForFlow = (ops: Operation[]): FlowDisplayInfo => {
  ops.sort((a, b) => Number(a.id - b.id));
  const firstOp = ops[0];

  const info: FlowDisplayInfo = {
    flowID: firstOp.flowId,
    planID: firstOp.planId,
    repoID: firstOp.repoId,
    snapshotID: firstOp.snapshotId,
    instanceID: firstOp.instanceId,
    type: getTypeForDisplay(firstOp),
    status: firstOp.status,
    displayTime: Number(firstOp.unixTimeStartMs),
    subtitleComponents: [],
    hidden: false,
    operations: [...ops], // defensive copy
  };

  const duration = Number(firstOp.unixTimeEndMs - firstOp.unixTimeStartMs);

  if (firstOp.status === OperationStatus.STATUS_PENDING) {
    info.subtitleComponents.push(m.op_subtitle_scheduled());
  }

  switch (firstOp.op.case) {
    case "operationBackup":
      {
        const lastStatus = firstOp.op.value.lastStatus;
        if (lastStatus) {
          if (lastStatus.entry.case === "status") {
            const percentage = lastStatus.entry.value.percentDone * 100;
            const bytesDone = formatBytes(Number(lastStatus.entry.value.bytesDone));
            const totalBytes = formatBytes(Number(lastStatus.entry.value.totalBytes));
            info.subtitleComponents.push(m.op_subtitle_processed({ percentage: percentage.toFixed(2) }));
            info.subtitleComponents.push(m.op_subtitle_bytes_total({ bytes: bytesDone, total: totalBytes }));
          } else if (lastStatus.entry.case === "summary") {
            const totalBytes = formatBytes(Number(lastStatus.entry.value.totalBytesProcessed));
            info.subtitleComponents.push(m.op_subtitle_summary({ bytes: totalBytes, duration: formatDuration(duration) }));
            info.subtitleComponents.push(m.op_subtitle_id({ id: normalizeSnapshotId(lastStatus.entry.value.snapshotId) }));
          }
        }
      }
      break;
    case "operationRestore":
      {
        const lastStatus = firstOp.op.value.lastStatus;
        if (lastStatus) {
          if (lastStatus.messageType === "summary") {
            const totalBytes = formatBytes(Number(lastStatus.totalBytes));
            info.subtitleComponents.push(m.op_subtitle_summary({ bytes: totalBytes, duration: formatDuration(duration) }));
          } else if (lastStatus.messageType === "status") {
            const percentage = lastStatus.percentDone * 100;
            const bytesDone = formatBytes(Number(lastStatus.bytesRestored));
            const totalBytes = formatBytes(Number(lastStatus.totalBytes));
            info.subtitleComponents.push(m.op_subtitle_processed({ percentage: percentage.toFixed(2) }));
            info.subtitleComponents.push(m.op_subtitle_bytes_total({ bytes: bytesDone, total: totalBytes }));
          }
        }
        info.subtitleComponents.push(m.op_subtitle_id({ id: normalizeSnapshotId(firstOp.snapshotId) }));
      }
      break;
    case "operationIndexSnapshot":
      const snapshot = firstOp.op.value.snapshot;
      if (!snapshot) break;
      if (snapshot.summary && snapshot.summary.totalBytesProcessed) {
        info.subtitleComponents.push(m.op_subtitle_summary({ bytes: formatBytes(Number(snapshot.summary.totalBytesProcessed)), duration: formatDuration(snapshot.summary.totalDuration * 1000) }));
      }
      info.subtitleComponents.push(m.op_subtitle_id({ id: normalizeSnapshotId(snapshot.id) }));
      break;
    default:
      switch (firstOp.status) {
        case OperationStatus.STATUS_INPROGRESS:
          info.subtitleComponents.push(m.op_subtitle_running());
          break;
        case OperationStatus.STATUS_USER_CANCELLED:
          info.subtitleComponents.push(m.op_subtitle_cancelled_user());
          break;
        case OperationStatus.STATUS_SYSTEM_CANCELLED:
          info.subtitleComponents.push(m.op_subtitle_cancelled_system());
          break;
        default:
          if (duration > 100) {
            info.subtitleComponents.push(m.op_subtitle_took({ duration: formatDuration(duration) }));
          }
          break;
      }
  }

  for (let op of ops) {
    if (op.op.case === "operationIndexSnapshot") {
      if (op.op.value.forgot) {
        info.hidden = true;
      }
    }
    if (op.op.case === "operationRunHook" && op.status === OperationStatus.STATUS_ERROR) {
      if (info.status === OperationStatus.STATUS_SUCCESS) {
        info.status = OperationStatus.STATUS_WARNING;
      }
    } else if (op.status === OperationStatus.STATUS_INPROGRESS || op.status === OperationStatus.STATUS_ERROR || op.status === OperationStatus.STATUS_WARNING) {
      info.status = op.status;
    }
  }

  return info;
}

export const shouldHideOperation = (operation: Operation) => {
  return (
    operation.op.case === "operationStats" ||
    shouldHideStatus(operation.status)
  );
};
export const shouldHideStatus = (status: OperationStatus) => {
  return status === OperationStatus.STATUS_SYSTEM_CANCELLED;
};

export const getTypeForDisplay = (op: Operation) => {
  switch (op.op.case) {
    case "operationBackup":
      return DisplayType.BACKUP;
    case "operationIndexSnapshot":
      return DisplayType.SNAPSHOT;
    case "operationForget":
      return DisplayType.FORGET;
    case "operationPrune":
      return DisplayType.PRUNE;
    case "operationCheck":
      return DisplayType.CHECK;
    case "operationRestore":
      return DisplayType.RESTORE;
    case "operationStats":
      return DisplayType.STATS;
    case "operationRunHook":
      return DisplayType.RUNHOOK;
    case "operationRunCommand":
      return DisplayType.RUNCOMMAND;
    default:
      return DisplayType.UNKNOWN;
  }
};

export const displayTypeToString = (type: DisplayType) => {
  switch (type) {
    case DisplayType.BACKUP:
      return m.op_type_backup();
    case DisplayType.SNAPSHOT:
      return m.op_type_snapshot();
    case DisplayType.FORGET:
      return m.op_type_forget();
    case DisplayType.PRUNE:
      return m.op_type_prune();
    case DisplayType.CHECK:
      return m.op_type_check();
    case DisplayType.RESTORE:
      return m.op_type_restore();
    case DisplayType.STATS:
      return m.op_type_stats();
    case DisplayType.RUNHOOK:
      return m.op_type_run_hook();
    case DisplayType.RUNCOMMAND:
      return m.op_type_run_command();
    default:
      return m.op_type_unknown();
  }
};

export const colorForStatus = (status: OperationStatus) => {
  switch (status) {
    case OperationStatus.STATUS_PENDING:
      return "grey";
    case OperationStatus.STATUS_INPROGRESS:
      return "blue";
    case OperationStatus.STATUS_ERROR:
      return "red";
    case OperationStatus.STATUS_WARNING:
      return "orange";
    case OperationStatus.STATUS_SUCCESS:
      return "green";
    case OperationStatus.STATUS_USER_CANCELLED:
      return "orange";
    default:
      return "grey";
  }
};

export const nameForStatus = (status: OperationStatus) => {
  switch (status) {
    case OperationStatus.STATUS_PENDING:
      return m.op_status_pending();
    case OperationStatus.STATUS_INPROGRESS:
      return m.op_status_in_progress();
    case OperationStatus.STATUS_ERROR:
      return m.op_status_error();
    case OperationStatus.STATUS_WARNING:
      return m.op_status_warning();
    case OperationStatus.STATUS_SUCCESS:
      return m.op_status_success();
    case OperationStatus.STATUS_USER_CANCELLED:
      return m.op_status_cancelled();
    case OperationStatus.STATUS_SYSTEM_CANCELLED:
      return m.op_status_cancelled();
    default:
      return m.op_status_unknown();
  }
}