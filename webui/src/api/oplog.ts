import {
  Operation,
  OperationEvent,
  OperationEventSchema,
  OperationStatus,
} from "../../gen/ts/v1/operations_pb";
import { GetOperationsRequest } from "../../gen/ts/v1/service_pb";
import { fromBinary, toBinary } from "@bufbuild/protobuf";
import { backrestService } from "./client";
import { createSharedStream } from "./streams/sharedStream";

// Operation-event stream, shared across tabs (see sharedStream). onResync means
// reset + refetch; consumers load their own initial state via getOperations().
export const operationsStream = createSharedStream<OperationEvent>({
  name: "backrest:operations",
  connect: (signal) => backrestService.getOperationEvents({}, { signal }),
  encode: (event) => toBinary(OperationEventSchema, event),
  decode: (bytes) => fromBinary(OperationEventSchema, bytes),
});

export const getOperations = async (
  req: GetOperationsRequest,
): Promise<Operation[]> => {
  const opList = await backrestService.getOperations(req);
  return opList.operations || [];
};

export const shouldHideOperation = (operation: Operation) => {
  // Hide successful backups with no snapshot ID (e.g., --skip-if-unchanged)
  // but NOT dry run backups which intentionally have no snapshot
  const isSkippedBackup =
    operation.status === OperationStatus.STATUS_SUCCESS &&
    operation.op.case === "operationBackup" &&
    !operation.snapshotId &&
    !operation.op.value.dryRun;

  return (
    operation.op.case === "operationStats" ||
    isSkippedBackup ||
    shouldHideStatus(operation.status)
  );
};
export const shouldHideStatus = (status: OperationStatus) => {
  return status === OperationStatus.STATUS_SYSTEM_CANCELLED;
};
