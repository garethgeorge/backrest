import React from "react";
import { EOperation, getOperations } from "../state/oplog";
import { Tree } from "antd";
import _ from "lodash";
import { DataNode } from "antd/es/tree";
import { formatDate, formatTime } from "../lib/formatting";

export const OperationTree = ({
  operations,
}: React.PropsWithoutRef<{ operations: EOperation[] }>) => {
  operations.sort((a, b) => b.parsedTime - a.parsedTime);
  return <Tree treeData={buildTree(operations)}></Tree>;
};

// TODO: more work on this view
const buildTree = (operations: EOperation[]): DataNode[] => {
  const grouped = _.groupBy(operations, (op) => {
    return new Date(op.parsedTime).toLocaleDateString("default", {
      month: "long",
      year: "numeric",
      day: "numeric",
    });
  });

  return _.keys(grouped).map((key) => {
    return {
      key: key,
      title: key,
      children: grouped[key].map((op) => {
        return {
          key: op.id!,
          title: <span>{formatTime(op.parsedTime)} - AN OPERATION</span>,
        };
      }),
    };
  });
};
