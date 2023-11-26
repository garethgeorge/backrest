import React from "react";
import { EOperation, getOperations } from "../state/oplog";
import { Tree } from "antd";
import _ from "lodash";
import { DataNode } from "antd/es/tree";
import { formatDate, formatTime } from "../lib/formatting";

export const OperationTree = ({
  operations,
}: React.PropsWithoutRef<{ operations: EOperation[] }>) => {
  operations = [...operations].reverse(); // reverse such that newest operations are at index 0.

  const treeData = buildTreeYear(operations);
  const keys = buildDefaultExpandedKeys(treeData);

  return <Tree treeData={treeData} defaultExpandedKeys={keys}></Tree>;
};

const buildDefaultExpandedKeys = (tree?: DataNode[]) => {
  const keys: string[] = [];
  while (tree && tree.length > 0) {
    const node = tree[0];
    keys.push(node.key as string);
    tree = node.children!;
  }
  return keys;
};

const buildTreeYear = (operations: EOperation[]): DataNode[] => {
  const grouped = _.groupBy(operations, (op) => {
    return op.parsedDate.getFullYear();
  });

  const entries: DataNode[] = _.map(grouped, (value, key) => {
    return {
      key: "y" + key,
      title: "" + key,
      children: buildTreeMonth(value),
    };
  });
  return entries;
};

const buildTreeMonth = (operations: EOperation[]): DataNode[] => {
  const grouped = _.groupBy(operations, (op) => {
    return op.parsedDate.getMonth();
  });
  const entries: DataNode[] = _.map(grouped, (value, key) => {
    return {
      key: "m" + key,
      title: value[0].parsedDate.toLocaleString("default", {
        month: "long",
        year: "numeric",
      }),
      children: buildTreeDay(value),
    };
  });
  return entries;
};

const buildTreeDay = (operations: EOperation[]): DataNode[] => {
  const grouped = _.groupBy(operations, (op) => {
    console.log("Operation date: " + formatDate(op.parsedTime));
    return formatDate(op.parsedTime);
  });

  const entries = _.map(grouped, (value, key) => {
    return {
      key: "d" + key,
      title: formatDate(value[0].parsedTime),
      children: buildTreeLeaf(value),
    };
  });
  return entries;
};

const buildTreeLeaf = (operations: EOperation[]): DataNode[] => {
  return _.map(operations, (op) => {
    return {
      key: op.id!,
      title: formatTime(op.parsedDate) + " - ",
      isLeaf: true,
    };
  });
};
