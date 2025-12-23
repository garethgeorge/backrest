"use client"

import { Menu as ChakraMenu, Portal } from "@chakra-ui/react"
import * as React from "react"

export interface MenuContentProps extends ChakraMenu.ContentProps {
  portalled?: boolean
  portalRef?: React.RefObject<HTMLElement | null>
}

export const MenuContent = React.forwardRef<HTMLDivElement, MenuContentProps>(
  function MenuContent(props, ref) {
    const { portalled = true, portalRef, ...rest } = props
    return (
      <Portal disabled={!portalled} container={portalRef}>
        {/* @ts-ignore */}
        <ChakraMenu.Positioner>
          <ChakraMenu.Content ref={ref} {...rest} />
        </ChakraMenu.Positioner>
      </Portal>
    )
  },
)

export const MenuRoot = ChakraMenu.Root
export const MenuTrigger = ChakraMenu.Trigger
export const MenuItem = ChakraMenu.Item
export const MenuItemText = ChakraMenu.ItemText
export const MenuItemCommand = ChakraMenu.ItemCommand
export const MenuSeparator = ChakraMenu.Separator
