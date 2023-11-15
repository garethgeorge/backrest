import { atom } from "recoil";
import {
  Operation,
  OperationEvent,
  OperationEventType,
  OperationStatus,
} from "../../gen/ts/v1/operations.pb";
import { ResticUI } from "../../gen/ts/v1/service.pb";
import { EventEmitter } from "events";

export const operationEmitter = new EventEmitter();

// Start fetching and emitting operations.
(async () => {
  await ResticUI.GetOperationEvents(
    {},
    (event: OperationEvent) => {
      operationEmitter.emit("operation", event);
    },
    {
      pathPrefix: "/api",
    }
  );
})();
