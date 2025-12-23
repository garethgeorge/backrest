"use client"

import { Combobox as ChakraCombobox, Portal } from "@chakra-ui/react"
import { CloseButton } from "./close-button"
import * as React from "react"

interface ComboboxControlProps extends ChakraCombobox.ControlProps {
  clearable?: boolean
  children?: React.ReactNode
}

export const ComboboxControl = React.forwardRef<
  HTMLDivElement,
  ComboboxControlProps
>(function ComboboxControl(props, ref) {
  const { children, clearable, ...rest } = props
  return (
    // @ts-ignore
    <ChakraCombobox.Control {...rest} ref={ref}>
      {children}
      <ChakraCombobox.IndicatorGroup>
        {clearable && <ComboboxClearTrigger />}
        <ChakraCombobox.Trigger />
      </ChakraCombobox.IndicatorGroup>
    </ChakraCombobox.Control>
  )
})

const ComboboxClearTrigger = React.forwardRef<
  HTMLButtonElement,
  ChakraCombobox.ClearTriggerProps
>(function ComboboxClearTrigger(props, ref) {
  return (
    // @ts-ignore
    <ChakraCombobox.ClearTrigger asChild {...props} ref={ref}>
      <CloseButton
        size="xs"
        variant="plain"
        focusVisibleRing="inside"
        focusRingWidth="2px"
        pointerEvents="auto"
      />
    </ChakraCombobox.ClearTrigger>
  )
})

interface ComboboxContentProps extends ChakraCombobox.ContentProps {
  portalled?: boolean
  portalRef?: React.RefObject<HTMLElement | null>
  children?: React.ReactNode
}

export const ComboboxContent = React.forwardRef<
  HTMLDivElement,
  ComboboxContentProps
>(function ComboboxContent(props, ref) {
  const { portalled = true, portalRef, ...rest } = props
  return (
    <Portal disabled={!portalled} container={portalRef}>
      {/* @ts-ignore */}
      <ChakraCombobox.Positioner>
        <ChakraCombobox.Content {...rest} ref={ref} />
      </ChakraCombobox.Positioner>
    </Portal>
  )
})

export interface ComboboxItemProps extends ChakraCombobox.ItemProps {
    children?: React.ReactNode
    item: any
}

export const ComboboxItem = React.forwardRef<
  HTMLDivElement,
  ComboboxItemProps
>(function ComboboxItem(props, ref) {
  const { item, children, ...rest } = props
  return (
    // @ts-ignore
    <ChakraCombobox.Item key={item.value} item={item} {...rest} ref={ref}>
      {children}
      <ChakraCombobox.ItemIndicator />
    </ChakraCombobox.Item>
  )
})

export const ComboboxRoot = ChakraCombobox.Root

// @ts-ignore
export const ComboboxInput = React.forwardRef<HTMLInputElement, ChakraCombobox.InputProps & React.InputHTMLAttributes<HTMLInputElement>>((props, ref) => {
    return <ChakraCombobox.Input ref={ref} {...props} />
})

export const ComboboxTrigger = ChakraCombobox.Trigger
export const ComboboxPositioner = ChakraCombobox.Positioner
export const ComboboxItemGroup = ChakraCombobox.ItemGroup
export const ComboboxLabel = ChakraCombobox.Label
export const ComboboxItemText = ChakraCombobox.ItemText
export const ComboboxEmpty = ChakraCombobox.Empty
