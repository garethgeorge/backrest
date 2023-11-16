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
          pathPrefix: "/api",
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
}: GetOperationsRequest): Promise<Operation[]> => {
  const opList = await ResticUI.GetOperations(
    {
      planId,
      repoId,
      lastN,
    },
    {
      pathPrefix: "/api",
    }
  );
  return opList.operations || [];
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
  callback: (event: OperationEvent | null, list: Operation[]) => void
) => {
  let operations: Operation[] = [];

  (async () => {
    const opsFromServer = await getOperations(req);
    operations = opsFromServer.filter(
      (o) => !operations.find((op) => op.id === o.id)
    );
    operations.sort((a, b) => {
      return parseInt(a.id!) - parseInt(b.id!);
    });

    callback(null, operations);
  })();

  return (event: OperationEvent) => {
    const op = event.operation!;
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
          return parseInt(a.id!) - parseInt(b.id!);
        });
      }
    } else if (type === OperationEventType.EVENT_CREATED) {
      operations.push(op);
    }

    callback(event, operations);
  };
};

// OperationsStateTracker tracks the state of operations starting with an initial query
export class OperationListSubscriber {
  private listener: ((event: OperationEvent) => void) | null = null;
  private operations: Operation[] = [];
  private eventEmitter = new EventEmitter();
  constructor(private req: GetOperationsRequest) {
    this.listener = (event: OperationEvent) => {
      this.eventEmitter.emit("changed");
    };
    subscribeToOperations(this.listener);
    getOperations(req).then((ops) => {
      this.operations = ops;
      this.eventEmitter.emit("changed");
    });
  }

  getOperations() {
    return this.operations;
  }

  onChange(callback: () => void) {
    this.eventEmitter.on("changed", callback);
  }

  destroy() {
    unsubscribeFromOperations(this.listener!);
  }
}