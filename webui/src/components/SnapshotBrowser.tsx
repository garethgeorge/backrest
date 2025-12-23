import React, { useEffect, useMemo, useState } from "react";
import {
  ListSnapshotFilesRequestSchema,
  ListSnapshotFilesResponse,
  ListSnapshotFilesResponseSchema,
  LsEntry,
  LsEntrySchema,
  RestoreSnapshotRequest,
  RestoreSnapshotRequestSchema,
} from "../../gen/ts/v1/service_pb";
import { FiFile, FiFolder, FiDownload, FiInfo, FiRefreshCw, FiMoreHorizontal } from "react-icons/fi";
import { useShowModal } from "./ModalManager";
import { formatBytes, normalizeSnapshotId } from "../lib/formatting";
import { URIAutocomplete } from "./URIAutocomplete";
import { backrestService } from "../api";
import { ConfirmButton } from "./SpinButton";
import { pathSeparator } from "../state/buildcfg";
import { create, toJsonString } from "@bufbuild/protobuf";
import {
  createTreeCollection,
  Flex,
  Box,
  Text,
  Button,
  Stack,
  Spinner
} from "@chakra-ui/react";
import {
  TreeViewRoot,
  TreeViewTree,
  TreeViewNode,
  TreeViewItem,
  TreeViewBranchControl,
  TreeViewBranchText,
  TreeViewItemText,
  TreeViewBranchIndicator,
  TreeViewBranchContent,
  TreeViewBranch,
  TreeViewNodeProvider,
} from "./ui/tree-view";
import {
  MenuRoot,
  MenuTrigger,
  MenuContent,
  MenuItem,
  MenuItemText
} from "./ui/menu";
import { FormModal } from "./FormModal";
import { Field } from "./ui/field";
import { toaster } from "./ui/toaster";

const SnapshotBrowserContext = React.createContext<{
  snapshotId: string;
  planId?: string;
  repoId: string;
  showModal: (modal: React.ReactNode) => void;
} | null>(null);

interface SnapshotNode {
  key: string;
  title: React.ReactNode;
  children?: SnapshotNode[];
  isLeaf?: boolean;
  entry: LsEntry;
}

// replaceKeyInTree returns a value only if changes are made.
const replaceKeyInTree = (
  curNode: SnapshotNode,
  setKey: string,
  setValue: SnapshotNode
): SnapshotNode | null => {
  if (curNode.key === setKey) {
    return setValue;
  }
  if (!curNode.children || setKey.indexOf(curNode.key as string) === -1) {
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

const findInTree = (curNode: SnapshotNode, key: string): SnapshotNode | null => {
  if (curNode.key === key) {
    return curNode;
  }
  if (!curNode.children || key.indexOf(curNode.key as string) === -1) {
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
  repoGuid,
  planId,
  snapshotId,
  snapshotOpId,
}: React.PropsWithoutRef<{
  snapshotId: string;
  snapshotOpId?: bigint;
  repoId: string;
  repoGuid: string;
  planId?: string;
}>) => {
  const showModal = useShowModal();
  const [treeData, setTreeData] = useState<SnapshotNode[]>([]);
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(new Set());
  const [loadingKeys, setLoadingKeys] = useState<Set<string>>(new Set());

  const respToNodes = (resp: ListSnapshotFilesResponse): SnapshotNode[] => {
    return resp.entries!
      // Strictly filter children that are longer than the parent path to avoid self-references.
      // This is crucial for fixing the infinite recursion / display issues, while ensuring
      // we don't accidentally filter the root node in other contexts (though root is manually init now).
      .filter((entry) => entry.path!.length > resp.path!.length)
      .map((entry) => ({
        key: entry.path!,
        title: <FileNode entry={entry} snapshotOpId={snapshotOpId} />,
        isLeaf: entry.type === "file",
        children: entry.type !== "file" ? [] : undefined,
        entry: entry
      }));
  };

  useEffect(() => {
    // Manually initialize the root node, ensures tree always starts with a visible root.
    const rootEntry = create(LsEntrySchema, {
      path: "/",
      type: "directory",
      name: "/",
    });
    setTreeData([
      {
        key: rootEntry.path!,
        title: <FileNode entry={rootEntry} snapshotOpId={snapshotOpId} />,
        isLeaf: false,
        children: [],
        entry: rootEntry,
      },
    ]);

    // Auto-load root explicitly as an expanded key so it fetches children immediately
    // or just rely on the user expanding it?
    // Antd version initialized with root but didn't auto-expand?
    // Actually Antd version called loadData via user interaction usually.
    // But we can trigger a load for root immediately to populate it if expanded.
    loadData("/");
    setExpandedKeys(new Set(["/"])); // Auto-expand root for better UX
  }, [repoId, repoGuid, snapshotId]);

  const loadData = async (key: string) => {
    // Check if node exists first
    let exists = false;
    for (const node of treeData) {
      if (findInTree(node, key)) {
        exists = true;
        break;
      }
    }

    let path = key;
    if (!path.endsWith("/")) {
      path += "/";
    }

    setLoadingKeys((prev) => {
      const next = new Set(prev);
      next.add(key);
      return next;
    });

    try {
      const resp = await backrestService.listSnapshotFiles(
        create(ListSnapshotFilesRequestSchema, {
          path,
          repoGuid,
          snapshotId,
        })
      );

      setTreeData((prev) => {
        let toUpdate: SnapshotNode | null = null;
        for (const node of prev) {
          toUpdate = findInTree(node, key);
          if (toUpdate) break;
        }

        if (!toUpdate) return prev;

        const toUpdateCopy = { ...toUpdate };
        toUpdateCopy.children = respToNodes(resp);

        return prev.map((node) => {
          const didUpdate = replaceKeyInTree(node, key, toUpdateCopy);
          return didUpdate || node;
        });
      });
    } catch (e: any) {
      toaster.create({ description: "Failed to load snapshot files: " + e.message, type: "error" });
    } finally {
      setLoadingKeys((prev) => {
        const next = new Set(prev);
        next.delete(key);
        return next;
      });
    }
  };

  const collection = useMemo(() => {
    return createTreeCollection<SnapshotNode>({
      nodeToValue: (node: SnapshotNode) => node?.key ?? "",
      nodeToString: (node: SnapshotNode) => node?.key ?? "",

      rootNode: {
        key: "root",
        title: "root",
        children: treeData,
        entry: create(LsEntrySchema, {}),
        isLeaf: false
      }
    })
  }, [treeData]);



  return (
    <SnapshotBrowserContext.Provider
      value={{ snapshotId, repoId, planId, showModal }}
    >
      <Box minH="200px" overflow="auto">
        {/* @ts-ignore */}
        <TreeViewRoot
          collection={collection}
          expandedValue={Array.from(expandedKeys)}
        >
          <TreeViewTree>
            <TreeViewNode
              render={({ node, nodeState }: any) => {
                const isBranch = !!node.children;
                if (isBranch) {
                  return (
                    <TreeViewBranch>
                      <TreeViewBranchControl
                        cursor="pointer"
                        onClick={(e: any) => {
                          e.stopPropagation();
                          console.log("SnapshotBrowser onClick: " + node.key);
                          if (loadingKeys.has(node.key)) return;
                          const newExpanded = new Set(expandedKeys);
                          if (newExpanded.has(node.key)) {
                            newExpanded.delete(node.key);
                          } else {
                            newExpanded.add(node.key);
                            loadData(node.key);
                          }
                          setExpandedKeys(newExpanded);
                        }}
                      >
                        <Box mr={2}>
                          {loadingKeys.has(node.key) ? (
                            <Spinner size="xs" />
                          ) : (
                            <FiFolder />
                          )}
                        </Box>
                        <TreeViewBranchText>{node.title}</TreeViewBranchText>
                      </TreeViewBranchControl>
                      <TreeViewBranchContent />
                    </TreeViewBranch>
                  )
                }
                return (
                  // @ts-ignore
                  <TreeViewItem value={node.key}>
                    <Box mr={2}><FiFile /></Box>
                    <TreeViewItemText>{node.title}</TreeViewItemText>
                  </TreeViewItem>
                )
              }}
            />
          </TreeViewTree>
        </TreeViewRoot>
      </Box>
    </SnapshotBrowserContext.Provider>
  );
};

const FileNode = ({
  entry,
  snapshotOpId,
}: {
  entry: LsEntry;
  snapshotOpId?: bigint;
}) => {
  const { snapshotId, repoId, planId, showModal } = React.useContext(
    SnapshotBrowserContext
  )!;

  const doDownload = () => {
    backrestService
      .getDownloadURL({ opId: snapshotOpId!, filePath: entry.path! })
      .then((resp) => {
        window.open(resp.value, "_blank");
      })
      .catch((e) => {
        toaster.create({ description: "Failed to fetch download URL: " + e.message, type: "error" });
      });
  };

  const showInfo = () => {
    showModal(
      <FormModal
        title={"Path Info for " + entry.path}
        isOpen={true}
        onClose={() => showModal(null)}
        footer={null}
      >
        <Box as="pre" overflow="auto" p={2} bg="bg.muted" borderRadius="md">
          {toJsonString(LsEntrySchema, entry, {
            prettySpaces: 2,
          })}
        </Box>
      </FormModal>
    );
  };

  const restore = () => {
    showModal(
      <RestoreModal
        path={entry.path!}
        repoId={repoId}
        planId={planId}
        snapshotId={snapshotId}
      />
    );
  };

  return (
    <Flex align="center" justify="space-between" width="full">
      <Text>
        {entry.name}
        {entry.type === "file" && (
          <Text as="span" color="fg.muted" ml={2} fontSize="sm">
            ({formatBytes(Number(entry.size))})
          </Text>
        )}
      </Text>

      <Box onClick={(e: any) => e.stopPropagation()}>
        <MenuRoot>
          {/* @ts-ignore */}
          <MenuTrigger asChild>
            <Button size="xs" variant="ghost">
              <FiMoreHorizontal />
            </Button>
          </MenuTrigger>
          {/* @ts-ignore */}
          <MenuContent>
            {/* @ts-ignore */}
            <MenuItem value="info" onClick={showInfo}>
              <FiInfo />
              {/* @ts-ignore */}
              <MenuItemText>Info</MenuItemText>
            </MenuItem>
            {/* @ts-ignore */}
            <MenuItem value="restore" onClick={restore}>
              <FiRefreshCw />
              {/* @ts-ignore */}
              <MenuItemText>Restore to path</MenuItemText>
            </MenuItem>
            {snapshotOpId && (
              // @ts-ignore
              <MenuItem value="download" onClick={doDownload}>
                <FiDownload />
                {/* @ts-ignore */}
                <MenuItemText>Download</MenuItemText>
              </MenuItem>
            )}
          </MenuContent>
        </MenuRoot>
      </Box>
    </Flex>
  );
};

const RestoreModal = ({
  repoId,
  planId,
  snapshotId,
  path,
}: {
  repoId: string;
  planId?: string;
  snapshotId: string;
  path: string;
}) => {
  const showModal = useShowModal();
  const [target, setTarget] = useState("");
  const [error, setError] = useState<string | null>(null);

  const defaultPath = useMemo(() => {
    if (path === pathSeparator) {
      return "";
    }
    return path + "-backrest-restore-" + normalizeSnapshotId(snapshotId);
  }, [path]);

  useEffect(() => {
    setTarget(defaultPath);
  }, [defaultPath]);

  const handleValid = () => {
    // Basic validation
    if (target) {
      let p = target;
      if (p.endsWith(pathSeparator)) {
        p = p.slice(0, -1);
      }
      const dirname = basename(p);
    }
    return true;
  };

  const handleOk = async () => {
    try {
      await backrestService.restore(
        create(RestoreSnapshotRequestSchema, {
          repoId,
          planId,
          snapshotId,
          path,
          target,
        })
      );
      toaster.create({ description: "Restore started successfully.", type: "success" });
      showModal(null);
    } catch (e: any) {
      toaster.create({ description: "Failed to restore: " + e.message, type: "error" });
    }
  };

  return (
    <FormModal
      title={"Restore " + path}
      isOpen={true}
      onClose={() => showModal(null)}
      footer={
        <>
          <Button variant="ghost" onClick={() => showModal(null)}>Cancel</Button>
          <ConfirmButton
            onClickAsync={handleOk}
            confirmTitle="Confirm Restore?"
          >
            Restore
          </ConfirmButton>
        </>
      }
    >
      <Stack gap={4}>
        <Text>
          If restoring to a specific path, ensure that the path does not already
          exist or that you are comfortable overwriting the data at that
          location.
        </Text>
        <Text>
          You may set the path to an empty string to restore to your Downloads folder.
        </Text>

        <Field label="Restore to path" errorText={error}>
          <URIAutocomplete
            placeholder="Restoring to Downloads"
            value={target}
            onChange={(val: string) => setTarget(val || "")}
          />
        </Field>
      </Stack>
    </FormModal>
  );
};

const basename = (path: string) => {
  const idx = path.lastIndexOf(pathSeparator);
  if (idx === -1) {
    return path;
  }
  return path.slice(0, idx + 1);
};
