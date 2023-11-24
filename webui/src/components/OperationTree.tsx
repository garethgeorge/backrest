import React from "react";
import { EOperation, getOperations } from "../state/oplog";
import { Tree } from "antd";
import _ from "lodash";
import { DataNode } from "antd/es/tree";
import { formatDate, formatTime } from "../lib/formatting";
import {
  ExclamationOutlined,
  QuestionOutlined,
  SaveOutlined,
} from "@ant-design/icons";
import { OperationStatus } from "../../gen/ts/v1/operations.pb";

type OpTreeNode = DataNode & {
  operation?: EOperation;
};

export const OperationTree = ({
  operations,
}: React.PropsWithoutRef<{ operations: EOperation[] }>) => {
  operations = [...operations].reverse(); // reverse such that newest operations are at index 0.

  if (operations.length === 0) {
    return (
      <div>
        <QuestionOutlined /> No operations yet.
      </div>
    );
  }

  const treeData = buildTreeYear(operations);

  return (
    <Tree<OpTreeNode>
      treeData={treeData}
      showIcon
      defaultExpandedKeys={[operations[0].id!]}
      titleRender={(node: OpTreeNode): React.ReactNode => {
        if (node.title) {
          return <>{node.title}</>;
        }
        if (node.operation) {
          const op = node.operation;
          if (op.operationBackup) {
            return <>{formatTime(op.parsedDate)} - Backup Operation</>;
          } else if (op.operationIndexSnapshot) {
            return <>{formatTime(op.parsedDate)} - Index Snapshot</>;
          }
        }
        return <span>no associated title, no associated operation</span>;
      }}
    ></Tree>
  );
};

const buildTreeYear = (operations: EOperation[]): OpTreeNode[] => {
  const grouped = _.groupBy(operations, (op) => {
    return op.parsedDate.getFullYear();
  });

  const entries: OpTreeNode[] = _.map(grouped, (value, key) => {
    return {
      key: "y" + key,
      title: "" + key,
      children: buildTreeMonth(value),
    };
  });
  entries.sort(sortByKey);
  return entries;
};

const buildTreeMonth = (operations: EOperation[]): OpTreeNode[] => {
  const grouped = _.groupBy(operations, (op) => {
    return `y${op.parsedDate.getFullYear()}m${op.parsedDate.getMonth()}`;
  });
  const entries: OpTreeNode[] = _.map(grouped, (value, key) => {
    return {
      key: key,
      title: value[0].parsedDate.toLocaleString("default", {
        month: "long",
        year: "numeric",
      }),
      children: buildTreeDay(value),
    };
  });
  entries.sort(sortByKey);
  return entries;
};

const buildTreeDay = (operations: EOperation[]): OpTreeNode[] => {
  const grouped = _.groupBy(operations, (op) => {
    return `y${op.parsedDate.getFullYear()}m${op.parsedDate.getMonth()}d${op.parsedDate.getDate()}`;
  });

  const entries = _.map(grouped, (value, key) => {
    return {
      key: "d" + key,
      title: formatDate(value[0].parsedTime),
      children: buildTreeLeaf(value),
    };
  });
  entries.sort(sortByKey);
  return entries;
};

const buildTreeLeaf = (operations: EOperation[]): OpTreeNode[] => {
  const entries = _.map(operations, (op) => {
    let iconColor = "grey";
    let icon: React.ReactNode | null = <QuestionOutlined />;

    switch (op.status) {
      case OperationStatus.STATUS_SUCCESS:
        iconColor = "green";
        break;
      case OperationStatus.STATUS_ERROR:
        iconColor = "red";
        break;
      case OperationStatus.STATUS_INPROGRESS:
        iconColor = "blue";
        break;
    }

    if (op.status === OperationStatus.STATUS_ERROR) {
      icon = <ExclamationOutlined style={{ color: iconColor }} />;
    } else if (op.operationBackup) {
      icon = <SaveOutlined style={{ color: iconColor }} />;
    }

    return {
      key: op.id!,
      operation: op,
      icon: icon,
    };
  });
  entries.sort(sortByKey);
  return entries;
};

const sortByKey = (a: OpTreeNode, b: OpTreeNode) => {
  if (a.key < b.key) {
    return 1;
  } else if (a.key > b.key) {
    return -1;
  }
  return 0;
};
