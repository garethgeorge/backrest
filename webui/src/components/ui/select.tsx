"use client";

import type { CollectionItem } from "@chakra-ui/react";
import { Select as ChakraSelect, Portal } from "@chakra-ui/react";
import { CloseButton } from "./close-button";
import * as React from "react";

interface SelectTriggerProps extends ChakraSelect.ControlProps {
  clearable?: boolean;
  children?: React.ReactNode;
}

export const SelectTrigger = React.forwardRef<
  HTMLButtonElement,
  SelectTriggerProps
>(function SelectTrigger(props, ref) {
  const { children, clearable, ...rest } = props;
  return (
    <ChakraSelect.Control {...rest}>
      {/* @ts-ignore */}
      <ChakraSelect.Trigger ref={ref}>{children}</ChakraSelect.Trigger>
      {/* @ts-ignore */}
      <ChakraSelect.IndicatorGroup>
        {clearable && <SelectClearTrigger />}
        <ChakraSelect.Indicator />
      </ChakraSelect.IndicatorGroup>
    </ChakraSelect.Control>
  );
});

const SelectClearTrigger = React.forwardRef<
  HTMLButtonElement,
  ChakraSelect.ClearTriggerProps
>(function SelectClearTrigger(props, ref) {
  return (
    // @ts-ignore
    <ChakraSelect.ClearTrigger asChild {...props} ref={ref}>
      <CloseButton
        size="xs"
        variant="plain"
        focusVisibleRing="inside"
        focusRingWidth="2px"
        pointerEvents="auto"
      />
    </ChakraSelect.ClearTrigger>
  );
});

interface SelectContentProps extends ChakraSelect.ContentProps {
  portalled?: boolean;
  portalRef?: React.RefObject<HTMLElement | null>;
  children?: React.ReactNode;
}

export const SelectContent = React.forwardRef<
  HTMLDivElement,
  SelectContentProps
>(function SelectContent(props, ref) {
  const { portalled = true, portalRef, children, ...rest } = props;
  return (
    <Portal disabled={!portalled} container={portalRef}>
      {/* @ts-ignore */}
      <ChakraSelect.Positioner>
        {/* @ts-ignore */}
        <ChakraSelect.Content {...rest} ref={ref}>
          {children}
        </ChakraSelect.Content>
      </ChakraSelect.Positioner>
    </Portal>
  );
});

interface SelectItemProps extends ChakraSelect.ItemProps {
  item: any;
  children?: React.ReactNode;
}

export const SelectItem = React.forwardRef<HTMLDivElement, SelectItemProps>(
  function SelectItem(props, ref) {
    const { item, children, ...rest } = props;
    return (
      // @ts-ignore
      <ChakraSelect.Item item={item} {...rest} ref={ref}>
        {children}
        <ChakraSelect.ItemIndicator />
      </ChakraSelect.Item>
    );
  },
);

// @ts-ignore
export const SelectValueText = ChakraSelect.ValueText;
export const SelectRoot = ChakraSelect.Root;
export const SelectLabel = ChakraSelect.Label;
export const SelectIndicator = ChakraSelect.Indicator;
export const SelectItemGroup = ChakraSelect.ItemGroup;
export const SelectItemText = ChakraSelect.ItemText;
export const SelectHiddenSelect = ChakraSelect.HiddenSelect;
