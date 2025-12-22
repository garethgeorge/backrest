import {
  Operation,
  OperationEvent,
  OperationEventType,
  OperationStatus,
} from "../../gen/ts/v1/operations_pb";
import { GetOperationsRequest, OpSelector } from "../../gen/ts/v1/service_pb";
import { BackupProgressEntry, ResticSnapshot, RestoreProgressEntry } from "../../gen/ts/v1/restic_pb";
import { EmptySchema } from "../../gen/ts/types/value_pb";
import { create } from "@bufbuild/protobuf";
import { backrestService } from "../api";
import { useEffect, useState } from "react";

const subscribers: ((event?: OperationEvent, err?: Error) => void)[] = [];

// Start fetching and emitting operations.
(async () => {
  while (true) {
    let nextConnWaitUntil = new Date().getTime() + 5000;
    try {
      for await (const event of backrestService.getOperationEvents({})) {
        console.log("operation event", event);
        subscribers.forEach((subscriber) => subscriber(event, undefined));
      }
    } catch (e: any) {
      console.warn("operations stream died with exception: ", e);
    }
    await new Promise((accept, _) =>
      setTimeout(accept, nextConnWaitUntil - new Date().getTime()),
    );
    subscribers.forEach((subscriber) => subscriber(undefined, new Error("reconnecting")));
  }
})();

export const getOperations = async (
  req: GetOperationsRequest,
): Promise<Operation[]> => {
  const opList = await backrestService.getOperations(req);
  return opList.operations || [];
};

export const subscribeToOperations = (
  callback: (event?: OperationEvent, err?: Error) => void,
) => {
  subscribers.push(callback);
  console.log("subscribed to operations, subscriber count: ", subscribers.length);
};

export const unsubscribeFromOperations = (
  callback: (event?: OperationEvent, err?: Error) => void,
) => {
  const index = subscribers.indexOf(callback);
  if (index > -1) {
    subscribers[index] = subscribers[subscribers.length - 1];
    subscribers.pop();
  }
  console.log("unsubscribed from operations, subscriber count: ", subscribers.length);
};

export const shouldHideOperation = (operation: Operation) => {
  return (
    operation.op.case === "operationStats" ||
    (operation.status === OperationStatus.STATUS_SUCCESS && operation.op.case === "operationBackup" && !operation.snapshotId) ||
    shouldHideStatus(operation.status)
  );
};
export const shouldHideStatus = (status: OperationStatus) => {
  return status === OperationStatus.STATUS_SYSTEM_CANCELLED;
};
