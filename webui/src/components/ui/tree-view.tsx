import { TreeView as ChakraTreeView } from "@chakra-ui/react"
import * as React from "react"

export const TreeViewRoot = ChakraTreeView.Root
export const TreeViewTree = ChakraTreeView.Tree
export const TreeViewNode = ChakraTreeView.Node
export const TreeViewItem = ChakraTreeView.Item
export const TreeViewLabel = ChakraTreeView.Label
export const TreeViewBranchControl = ChakraTreeView.BranchControl
export const TreeViewBranchText = ChakraTreeView.BranchText
export const TreeViewItemText = ChakraTreeView.ItemText
export const TreeViewBranchIndentGuide = ChakraTreeView.BranchIndentGuide
export const TreeViewBranchContent = ChakraTreeView.BranchContent
export const TreeViewBranchTrigger = ChakraTreeView.BranchTrigger
export const TreeViewBranchIndicator = ChakraTreeView.BranchIndicator
export const TreeViewBranch = ChakraTreeView.Branch
export const TreeViewNodeProvider = ChakraTreeView.NodeProvider

export interface TreeViewNodeProps<T = any> extends ChakraTreeView.NodeProps<T> {}
export interface TreeViewRootProps extends ChakraTreeView.RootProps {}
export interface TreeViewItemProps extends ChakraTreeView.ItemProps {}
