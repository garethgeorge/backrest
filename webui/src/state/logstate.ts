import { Operation, OperationEvent, OperationEventType, OperationStatus } from "../../gen/ts/v1/operations_pb";
import { GetOperationsRequest, GetOperationsRequestSchema, OpSelector } from "../../gen/ts/v1/service_pb";
import { getOperations, subscribeToOperations, unsubscribeFromOperations } from "./oplog";
import { create } from "@bufbuild/protobuf";

type Subscriber = (ids: bigint[], flowIDs: bigint[], event: OperationEventType) => void;

export const syncStateFromRequest = (state: OplogState, req: GetOperationsRequest, onError?: (e: Error) => void): () => void => {
  getOperations(req).then((res) => {
    state.add(...res);
  }).catch((e) => {
    if (onError) {
      onError(e);
    }
  });

  const cbHelper = (event?: OperationEvent, err?: Error) => {
    if (err) {
      if (onError) {
        onError(err);
      }
      state.reset();

      getOperations(req).then((res) => {
        state.add(...res);
      }).catch((e) => {
        if (onError) {
          onError(e);
        }
      });
    } else if (event) {
      switch (event.event.case) {
        case "createdOperations":
        case "updatedOperations":
          let ops = event.event.value.operations;
          if (req.selector) {
            ops = ops.filter((op) => matchSelector(req.selector!, op));
          }
          state.add(...ops);
          break;
        case "deletedOperations":
          state.removeIDs(...event.event.value.values);
          break;
      }
    }
  };

  subscribeToOperations(cbHelper);
  return () => { unsubscribeFromOperations(cbHelper); };
};


// getStatus returns the status of the last N operations that belong to a single snapshot.
const getStatus = async (req: GetOperationsRequest) => {
  let ops = await getOperations(req);
  ops.sort((a, b) => {
    return Number(b.unixTimeStartMs - a.unixTimeStartMs);
  });

  let flowID: BigInt | undefined = undefined;
  for (const op of ops) {
    if (
      op.status === OperationStatus.STATUS_PENDING ||
      op.status === OperationStatus.STATUS_SYSTEM_CANCELLED ||
      op.op.case === "operationRunHook"
    ) {
      continue;
    }
    if (!flowID) {
      flowID = op.flowId;
    } else if (flowID !== op.flowId) {
      break;
    }
    if (op.status !== OperationStatus.STATUS_SUCCESS) {
      return op.status;
    }
  }
  return OperationStatus.STATUS_SUCCESS;
};

export const getStatusForSelector = async (sel: OpSelector) => {
  const req = create(GetOperationsRequestSchema, {
    selector: sel,
    lastN: BigInt(20),
  });
  return await getStatus(req);
};

export class OplogState {
  private byID: Map<bigint, Operation> = new Map();
  private byFlowID: Map<bigint, Operation[]> = new Map();

  private subscribers: Set<Subscriber> = new Set();

  constructor(private filter: (op: Operation) => boolean = () => true) {
  }

  public subscribe(subscriber: Subscriber) {
    this.subscribers.add(subscriber);
  }

  public unsubscribe(subscriber: Subscriber) {
    this.subscribers.delete(subscriber);
  }

  public reset() {
    const idsRemoved = Array.from(this.byID.keys());
    const flowIDsRemoved = Array.from(this.byFlowID.keys());
    this.byID.clear();
    this.byFlowID.clear();

    for (let subscriber of this.subscribers) {
      subscriber(idsRemoved, flowIDsRemoved, OperationEventType.EVENT_DELETED);
    }
  }

  public getByFlowID(flowID: bigint): Operation[] | undefined {
    return this.byFlowID.get(flowID);
  }

  public getByID(id: bigint): Operation | undefined {
    return this.byID.get(id);
  }

  public getAll(): Operation[] {
    return Array.from(this.byID.values());
  }

  public add(...ops: Operation[]) {
    const idsRemoved: bigint[] = [];
    const ids: bigint[] = [];
    const flowIDsRemoved = new Set<bigint>();
    const flowIDs = new Set<bigint>();
    for (let op of ops) {
      if (!this.filter(op)) {
        idsRemoved.push(op.id);
        flowIDsRemoved.add(op.flowId);
        this.removeHelper(op);
      } else {
        const flow = this.byFlowID.get(op.flowId);
        if (op.op.case === "operationIndexSnapshot" && flow &&
          flow.find((o) => o.id !== op.id && o.op.case === "operationIndexSnapshot" && o.snapshotId === op.snapshotId)) {
          // don't add a second index snapshot for the same flow, this can happen in multihost mode
          continue;
        }

        ids.push(op.id);
        flowIDs.add(op.flowId);
        this.addHelper(op);
      }
    }

    if (idsRemoved.length > 0) {
      for (let subscriber of this.subscribers) {
        subscriber(idsRemoved, Array.from(flowIDsRemoved), OperationEventType.EVENT_DELETED);
      }
    }
    if (ids.length > 0) {
      for (let subscriber of this.subscribers) {
        subscriber(ids, Array.from(flowIDs), OperationEventType.EVENT_CREATED);
      }
    }
  }

  public removeIDs(...ids: bigint[]) {
    const ops: Operation[] = [];
    for (let id of ids) {
      let op = this.byID.get(id);
      if (op) {
        ops.push(op);
      }
    }
    this.remove(...ops);
  }

  public remove(...ops: Operation[]) {
    const ids: bigint[] = [];
    const flowIDs = new Set<bigint>();
    for (let op of ops) {
      ids.push(op.id);
      flowIDs.add(op.flowId);
      this.removeHelper(op);
    }

    for (let subscriber of this.subscribers) {
      subscriber(ids, Array.from(flowIDs), OperationEventType.EVENT_DELETED);
    }
  }

  private addHelper(op: Operation) {
    this.byID.set(op.id, op);
    let ops = this.byFlowID.get(op.flowId);
    if (!ops) {
      ops = [];
      this.byFlowID.set(op.flowId, ops);
    }
    let index = ops.findIndex((o) => o.id === op.id);
    if (index !== -1) {
      ops[index] = op;
    } else {
      ops.push(op);
    }
  }

  private removeHelper(op: Operation) {
    this.byID.delete(op.id);
    let ops = this.byFlowID.get(op.flowId);
    if (ops) {
      let index = ops.indexOf(op);
      if (index !== -1) {
        ops.splice(index, 1);
      }
    }
  }
}



// Defining matchers for each field in OpSelector to determine if an operation matches the selector.
// Type system asserts that a check must exist for each field in OpSelector.
const selectorFieldMatchers: { [K in keyof OpSelector]: (op: Operation, sel: OpSelector) => boolean } = {
  planId: (op, sel) => op.planId === sel.planId,
  repoGuid: (op, sel) => op.repoGuid === sel.repoGuid,
  flowId: (op, sel) => op.flowId === sel.flowId,
  instanceId: (op, sel) => op.instanceId === sel.instanceId,
  snapshotId: (op, sel) => op.snapshotId === sel.snapshotId,
  originalInstanceKeyid: (op, sel) => op.originalInstanceKeyid === sel.originalInstanceKeyid,
  ids: (op: Operation, sel: OpSelector) => sel.ids.length === 0 || sel.ids.includes(op.id),
  ["$typeName"]: (op: Operation, sel: OpSelector): boolean => true, // $typeName is a proto property that isn't used for matching
};

export const matchSelector = (selector: OpSelector, op: Operation): boolean => {
  for (const key in selector) {
    const matcher = selectorFieldMatchers[key as keyof OpSelector];
    if (matcher && !matcher(op, selector)) {
      return false;
    }
  }
  return true;
};
