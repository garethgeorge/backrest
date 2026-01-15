import React, { useEffect, useState } from "react";
import { Operation } from "../../../gen/ts/v1/operations_pb";
import {
  GetOperationsRequestSchema,
  type GetOperationsRequest,
} from "../../../gen/ts/v1/service_pb";
import { alerts } from "../../components/common/Alerts";
import { OperationRow } from "./OperationRow";
import { OplogState, syncStateFromRequest } from "../../api/logState";
import { shouldHideStatus } from "../../api/oplog";
import { toJsonString } from "@bufbuild/protobuf";
import { Stack, Box, Flex, Center, Spinner } from "@chakra-ui/react";
import { EmptyState } from "../../components/ui/empty-state";
import { FiList } from "react-icons/fi";
import {
  PaginationRoot,
  PaginationItems,
  PaginationNextTrigger,
  PaginationPrevTrigger,
} from "../../components/ui/pagination";

// OperationList displays a list of operations that are either fetched based on 'req' or passed in via 'useBackups'.
// If showPlan is provided the planId will be displayed next to each operation in the operation list.
export const OperationListView = ({
  req,
  useOperations,
  showPlan,
  displayHooksInline,
  filter,
  showDelete,
}: React.PropsWithoutRef<{
  req?: GetOperationsRequest;
  useOperations?: Operation[]; // exact set of operations to display; no filtering will be applied.
  showPlan?: boolean;
  displayHooksInline?: boolean;
  filter?: (op: Operation) => boolean;
  showDelete?: boolean; // allows deleting individual operation rows, useful for the list view in the plan / repo panels.
}>) => {
  const [operations, setOperations] = useState<Operation[]>([]);
  const [loading, setLoading] = useState(!!req);
  const [page, setPage] = useState(1);
  const pageSize = 25;

  useEffect(() => {
    if (!req) return;
    setLoading(true);
    const logState = new OplogState(
      (op) => !shouldHideStatus(op.status) && (!filter || filter(op)),
    );

    logState.subscribe((ids, flowIDs, event) => {
      const ops = logState.getAll();
      setOperations(ops);
      setLoading(false);
    });

    return syncStateFromRequest(logState, req, (e) => {
      alerts.error("Failed to fetch operations: " + e.message);
      setLoading(false);
    });
  }, [req ? toJsonString(GetOperationsRequestSchema, req) : ""]);

  const hookExecutionsForOperation: Map<bigint, Operation[]> = new Map();
  let operationsForDisplay: Operation[] = [];

  // Local variable to hold the source operations
  let sourceOperations = operations;
  if (useOperations) {
    sourceOperations = [...useOperations];
  }

  if (!displayHooksInline) {
    operationsForDisplay = sourceOperations.filter((op) => {
      if (op.op.case === "operationRunHook") {
        const parentOp = op.op.value.parentOp;
        if (!hookExecutionsForOperation.has(parentOp)) {
          hookExecutionsForOperation.set(parentOp, []);
        }
        hookExecutionsForOperation.get(parentOp)!.push(op);
        return false;
      }
      return true;
    });
  } else {
    operationsForDisplay = sourceOperations;
  }

  operationsForDisplay.sort((a, b) => {
    return Number(b.unixTimeStartMs - a.unixTimeStartMs);
  });

  if (!operationsForDisplay || operationsForDisplay.length === 0) {
    if (loading) {
      return (
        <Center py={8}>
          <Spinner />
        </Center>
      );
    }
    return (
      <EmptyState
        title="No operations yet"
        description="Operations will appear here once they start."
        icon={<FiList />}
      />
    );
  }

  const total = operationsForDisplay.length;
  const start = (page - 1) * pageSize;
  const pagedOperations = operationsForDisplay.slice(start, start + pageSize);

  return (
    <Stack gap={4}>
      {pagedOperations.map((op) => (
        <OperationRow
          key={op.id}
          operation={op}
          showPlan={showPlan || false}
          hookOperations={hookExecutionsForOperation.get(op.id)}
          showDelete={showDelete}
        />
      ))}

      {total > pageSize && (
        <Flex justify="center" mt={4}>
          <PaginationRoot
            count={total}
            pageSize={pageSize}
            page={page}
            onPageChange={(e: any) => setPage(e.page)}
          >
            <PaginationPrevTrigger />
            <PaginationItems />
            <PaginationNextTrigger />
          </PaginationRoot>
        </Flex>
      )}
    </Stack>
  );
};
