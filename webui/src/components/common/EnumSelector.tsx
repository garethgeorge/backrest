import React from "react";
import {
  SelectRoot,
  SelectTrigger,
  SelectValueText,
  SelectContent,
  SelectItem,
  SelectHiddenSelect,
} from "../ui/select";
import { Box, Text, createListCollection } from "@chakra-ui/react";

export interface EnumOption<T extends string> {
  value: T;
  label?: string;
  description?: string;
}

export interface EnumSelectorProps<T extends string> {
  value: T | T[];
  onChange: (value: T | T[]) => void;
  options: EnumOption<T>[];
  multiSelect?: boolean;
  placeholder?: string;
  size?: "xs" | "sm" | "md" | "lg";
}

export const EnumSelector = <T extends string>({
  value,
  onChange,
  options,
  multiSelect = false,
  placeholder = "Select...",
  size = "sm",
}: EnumSelectorProps<T>) => {
  const collection = createListCollection({
    items: options.map((opt) => ({
      label: opt.label || opt.value,
      value: opt.value,
      description: opt.description,
    })),
  });

  const handleValueChange = (e: { value: string[] }) => {
    if (multiSelect) {
      onChange(e.value as T[]);
    } else {
      onChange(e.value[0] as T);
    }
  };

  const currentValue = Array.isArray(value) ? value : value ? [value] : [];

  return (
    <SelectRoot
      multiple={multiSelect}
      collection={collection}
      value={currentValue}
      onValueChange={handleValueChange}
      size={size}
    >
      <SelectHiddenSelect />
      <SelectTrigger>
        <SelectValueText placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent zIndex={2000}>
        {collection.items.map((item: any) => (
          <SelectItem item={item} key={item.value}>
            <Box>
              <Text as="span">{item.label}</Text>
              {item.description && (
                <Text as="span" color="fg.muted" ml={2} fontSize="xs">
                  - {item.description}
                </Text>
              )}
            </Box>
          </SelectItem>
        ))}
      </SelectContent>
    </SelectRoot>
  );
};
