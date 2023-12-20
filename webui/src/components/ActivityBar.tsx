import React, { useEffect, useState } from "react";
import { EOperation, detailsForOperation, displayTypeToString, getTypeForDisplay, subscribeToOperations, toEop, unsubscribeFromOperations } from "../state/oplog";
import { OperationEvent, OperationStatus } from "../../gen/ts/v1/operations.pb";
import { formatDuration } from "../lib/formatting";

export const ActivityBar = () => {
    const [activeOperations, setActiveOperations] = useState<EOperation[]>([]);

    useEffect(() => {
        const callback = ({ operation, type }: OperationEvent) => {
            if (!operation || !type) return;

            setActiveOperations((ops) => {
                ops = ops.filter((op) => op.id !== operation.id);
                if (operation.status === OperationStatus.STATUS_INPROGRESS) {
                    ops.push(toEop(operation));
                }
                ops.sort((a, b) => b.parsedTime - a.parsedTime);
                return ops;
            });
        }

        subscribeToOperations(callback);

        return () => {
            unsubscribeFromOperations(callback);
        }
    });

    const details = activeOperations.map((op) => {
        return {
            op: op,
            details: detailsForOperation(op),
            displayName: displayTypeToString(getTypeForDisplay(op)),
        }
    });

    return <span>{details.map(details => {
        return <>{details.displayName} in progress for plan {details.op.planId} to {details.op.repoId} for {formatDuration(details.details.duration)}</>
    })}</span>
}