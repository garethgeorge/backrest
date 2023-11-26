import React, { useEffect, useMemo, useState } from "react";
import { Input, Tree } from "antd";
import type { DataNode, EventDataNode } from "antd/es/tree";
import {
  ListSnapshotFilesResponse,
  LsEntry,
  ResticUI,
} from "../../gen/ts/v1/service.pb";
import { useAlertApi } from "./Alerts";
import { FileOutlined, FolderOutlined } from "@ant-design/icons";

type ELsEntry = LsEntry & { children?: ELsEntry[] };

// replaceKeyInTree returns a value only if changes are made.
const replaceKeyInTree = (
  curNode: DataNode,
  setKey: string,
  setValue: DataNode
): DataNode | null => {
  if (curNode.key === setKey) {
    return setValue;
  }
  if (setKey.indexOf(curNode.key as string) === -1) {
    return null;
  }
  if (!curNode.children) {
    return null;
  }
  for (const idx in curNode.children!) {
    const child = curNode.children![idx];
    const newChild = replaceKeyInTree(child, setKey, setValue);
    if (newChild) {
      const curNodeCopy = { ...curNode };
      curNodeCopy.children = [...curNode.children!];
      curNodeCopy.children[idx] = newChild;
      return curNodeCopy;
    }
  }
  return null;
};
const findInTree = (curNode: DataNode, key: string): DataNode | null => {
  if (curNode.key === key) {
    return curNode;
  }

  if (!curNode.children) {
    return null;
  }

  for (const child of curNode.children) {
    const found = findInTree(child, key);
    if (found) {
      return found;
    }
  }
  return null;
};

export const SnapshotBrowser = ({
  repoId,
  snapshotId,
}: React.PropsWithoutRef<{ snapshotId: string; repoId: string }>) => {
  const alertApi = useAlertApi();
  const [treeData, setTreeData] = useState<DataNode[]>([]);

  useEffect(() => {
    (async () => {
      try {
        const resp = await ResticUI.ListSnapshotFiles(
          {
            path: "/",
            repoId,
            snapshotId,
          },
          { pathPrefix: "/api" }
        );
        setTreeData(respToNodes(resp));
      } catch (e: any) {
        alertApi?.error("Failed to list snapshot files: " + e.message);
      }
    })();
  }, [repoId, snapshotId]);

  const onLoadData = async ({ key, children }: EventDataNode<DataNode>) => {
    if (children) {
      return;
    }

    const resp = await ResticUI.ListSnapshotFiles(
      {
        path: (key + "/") as string,
        repoId,
        snapshotId,
      },
      { pathPrefix: "/api" }
    );

    let toUpdate: DataNode | null = null;
    for (const node of treeData) {
      toUpdate = findInTree(node, key as string);
      if (toUpdate) {
        break;
      }
    }

    if (!toUpdate) {
      return;
    }

    const toUpdateCopy = { ...toUpdate };
    toUpdateCopy.children = respToNodes(resp);

    const newTree = treeData.map((node) => {
      const didUpdate = replaceKeyInTree(node, key as string, toUpdateCopy);
      if (didUpdate) {
        return didUpdate;
      }
      return node;
    });

    setTreeData(newTree);
  };

  return <Tree<DataNode> loadData={onLoadData} treeData={treeData} />;
};

const respToNodes = (resp: ListSnapshotFilesResponse): DataNode[] => {
  const nodes = resp
    .entries!.filter((entry) => entry.path!.length > resp.path!.length)
    .map((entry) => {
      const lastSlash = entry.path!.lastIndexOf("/");
      const title =
        lastSlash === -1 ? entry.path : entry.path!.slice(lastSlash + 1);

      const node: DataNode = {
        key: entry.path!,
        title: title,
        isLeaf: entry.type === "file",
        icon: entry.type === "file" ? <FileOutlined /> : <FolderOutlined />,
      };
      return node;
    });

  console.log(JSON.stringify(nodes, null, 2));

  return nodes;
};
