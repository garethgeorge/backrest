import React, { useEffect, useMemo, useState } from "react";
import { Button, Dropdown, Input, Modal, Space, Tree } from "antd";
import type { DataNode, EventDataNode } from "antd/es/tree";
import {
  ListSnapshotFilesResponse,
  LsEntry,
  ResticUI,
} from "../../gen/ts/v1/service.pb";
import { useAlertApi } from "./Alerts";
import {
  DownOutlined,
  DownloadOutlined,
  FileOutlined,
  FolderOutlined,
  MenuFoldOutlined,
} from "@ant-design/icons";
import { drop } from "lodash";
import { useShowModal } from "./ModalManager";

const SnapshotBrowserContext = React.createContext<{
  snapshotId: string;
  repoId: string;
} | null>(null);

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

  return (
    <SnapshotBrowserContext.Provider value={{ snapshotId, repoId }}>
      <Tree<DataNode> loadData={onLoadData} treeData={treeData} />
    </SnapshotBrowserContext.Provider>
  );
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
        title: <FileNode entry={entry} />,
        isLeaf: entry.type === "file",
        icon: entry.type === "file" ? <FileOutlined /> : <FolderOutlined />,
      };
      return node;
    });

  console.log(JSON.stringify(nodes, null, 2));

  return nodes;
};

const FileNode = ({ entry }: { entry: LsEntry }) => {
  const [dropdown, setDropdown] = useState<React.ReactNode>(null);
  const { snapshotId, repoId } = React.useContext(SnapshotBrowserContext)!;

  return (
    <Space
      onMouseEnter={() =>
        setDropdown(
          <Dropdown
            menu={{
              items: [
                {
                  key: "restore",
                  label: "Restore to path",
                  onClick: () => {
                    restoreFlow(repoId, snapshotId, entry.path!);
                  },
                },
              ],
            }}
          >
            <DownloadOutlined />
          </Dropdown>
        )
      }
      onMouseLeave={() => setDropdown(null)}
    >
      {entry.name}
      {dropdown}
    </Space>
  );
};

const RestoreModal = ({
  repoId,
  snapshotId,
  path,
}: {
  repoId: string;
  snapshotId: string;
  path: string;
}) => {
  const showModal = useShowModal();
  const [confirmLoading, setConfirmLoading] = useState(false);
  const [restoreConfirmed, setRestoreConfirmed] = useState(false);

  const handleCancel = () => {
    showModal(null);
  };

  const handleOk = async () => {
    if (!restoreConfirmed) {
      setRestoreConfirmed(true);
      setTimeout(() => {
        setRestoreConfirmed(false);
      }, 2000);
      return;
    }

    setConfirmLoading(true);
    try {
      await ResticUI.RestoreSnapshot(
        {
          repoId,
          snapshotId,
          path,
        },
        { pathPrefix: "/api" }
      );
      setRestoreConfirmed(true);
    } catch (e: any) {
      alert("Failed to restore snapshot: " + e.message);
    } finally {
      setConfirmLoading(false);
    }
  };

  return (
    <Modal
      open={true}
      onCancel={handleCancel}
      title={
        "Restore " + path + " from snapshot " + snapshotId + " in " + repoId
      }
      width="40vw"
      footer={[
        <Button loading={confirmLoading} key="back" onClick={handleCancel}>
          Cancel
        </Button>,
        <Button
          key="submit"
          type="primary"
          loading={confirmLoading}
          onClick={handleOk}
        >
          {restoreConfirmed ? "Confirm Restore?" : "Restore"}
        </Button>,
      ]}
    ></Modal>
  );
};

const restoreFlow = (repoId: string, snapshotId: string, path: string) => {};
