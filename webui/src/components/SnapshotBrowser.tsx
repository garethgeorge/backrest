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
    TreeView,
    createTreeCollection,
    Flex,
    Box,
    Text,
    Button,
    Stack
} from "@chakra-ui/react";
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

  const respToNodes = (resp: ListSnapshotFilesResponse): SnapshotNode[] => {
    return resp.entries!
      .filter((entry) => entry.path!.length >= resp.path!.length)
      .map((entry) => ({
          key: entry.path!,
          title: <FileNode entry={entry} snapshotOpId={snapshotOpId} />,
          isLeaf: entry.type === "file",
          children: entry.type === "directory" ? [] : undefined, // Initialize empty children for dirs
          entry: entry
      }));
  };

  useEffect(() => {
    setTreeData(
      respToNodes(
        create(ListSnapshotFilesResponseSchema, {
          entries: [
            {
              path: "/",
              type: "directory",
              name: "/",
            },
          ],
          path: "/",
        })
      )
    );
     // Auto-load root?
     loadData("/");
  }, [repoId, repoGuid, snapshotId]);

  const loadData = async (key: string) => {
    // Find node, if children populated (and not empty array created initially??), skip?
    // Actually our initial creation makes empty children array.
    // Logic: check if we have fetched? 
    // We can assume if we call loadData we want to refresh or fetch.
    
    // Check if node exists first
    let exists = false;
    for (const node of treeData) {
        if (findInTree(node, key)) {
            exists = true;
            break;
        }
    }
    // If root (/) it might be the only node.
    
    let path = key;
    if (!path.endsWith("/")) {
      path += "/";
    }

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
    }
  };
  
 const collection = useMemo(() => {
     return createTreeCollection<SnapshotNode>({
         nodeToValue: (node: SnapshotNode) => node.key,
         nodeToString: (node: SnapshotNode) => node.key,
         rootNode: {
             key: "root",
             title: "root",
             children: treeData,
             entry: create(LsEntrySchema, {}),
             isLeaf: false
         } 
     })
  }, [treeData]);

  const handleExpandedChange = (e: any) => {
      // @ts-ignore
      const newExpanded = new Set(e.expandedIds as string[]);
      setExpandedKeys(newExpanded);
      
      // Find which key was added?
      // Or just iterate new keys and load if needed.
      // Easiest: checking which keys in newExpanded were NOT in expandedKeys
      for (const key of Array.from(newExpanded)) {
          if (!expandedKeys.has(key as string)) {
               // Newly expanded
               loadData(key as string);
          }
      }
  };

  return (
    <SnapshotBrowserContext.Provider
      value={{ snapshotId, repoId, planId, showModal }}
    >
       <Box minH="200px" overflow="auto">
        {/* @ts-ignore */}
        <TreeView.Root 
            collection={collection}
            // @ts-ignore
            expandedIds={Array.from(expandedKeys)}
            onExpandedChange={handleExpandedChange}
        >
            <TreeView.Tree>
                {/* @ts-ignore */}
                <TreeView.Node
                    render={({ node, nodeState }: any) => {
                         const isBranch = !!node.children;
                         if (isBranch) {
                             return (
                                 <TreeView.Branch>
                                    <TreeView.BranchControl>
                                        <Box mr={2}><FiFolder /></Box>
                                        <TreeView.BranchText>{node.title}</TreeView.BranchText>
                                        <TreeView.BranchIndicator>
                                            <FiInfo /> 
                                            {/* Note: Icon is managed in title? No title is component. */}
                                        </TreeView.BranchIndicator>
                                    </TreeView.BranchControl>
                                    <TreeView.BranchContent>
                                        <TreeView.NodeProvider />
                                    </TreeView.BranchContent>
                                 </TreeView.Branch>
                             )
                         }
                         return (
                             <TreeView.Item>
                                 <Box mr={2}><FiFile /></Box>
                                 <TreeView.ItemText>{node.title}</TreeView.ItemText>
                             </TreeView.Item>
                         )
                    }}
                />
            </TreeView.Tree>
        </TreeView.Root>
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
