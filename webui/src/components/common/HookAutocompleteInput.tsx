import React, { useState, useMemo, useRef, useEffect } from "react";
import { Input, Box } from "@chakra-ui/react";
import { backrestService } from "../../api/client";
import { StringList } from "../../../gen/ts/types/value_pb";

interface HookAutocompleteInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  action: string;
  field: string;
  inputProps?: any;
  allHooks?: any[];
}

export const HookAutocompleteInput = ({
  value,
  onChange,
  placeholder,
  action,
  field,
  inputProps,
  allHooks,
}: HookAutocompleteInputProps) => {
  const [items, setItems] = useState<string[]>([]);
  const [open, setOpen] = useState(false);
  const wrapperRef = useRef<HTMLDivElement>(null);

  const unsavedValues = useMemo(
    () =>
      Array.from(
        new Set(
          (allHooks || [])
            .map((h) => h[action]?.[field])
            .filter((v) => typeof v === "string" && v !== "")
        )
      ),
    [allHooks, action, field]
  );

  const filteredItems = useMemo(() => {
    const prefix = (value || "").toLowerCase();
    return Array.from(new Set([...items, ...unsavedValues]))
      .filter((i) => i !== value && i.toLowerCase().startsWith(prefix));
  }, [items, unsavedValues, value]);

  const doFetch = () => {
    backrestService
      .hookAutocomplete({ action, field })
      .then((res: StringList) => setItems(res.values || []))
      .catch((e) => console.error("Hook autocomplete error:", e));
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    onChange(e.target.value);
    setOpen(true);
  };

  const handleFocus = () => {
    setOpen(true);
    doFetch();
  };

  const selectItem = (item: string) => {
    onChange(item);
    setOpen(false);
  };

  useEffect(() => {
    if (!open) return;
    const handleClickOutside = (e: MouseEvent) => {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [open]);

  return (
    <Box ref={wrapperRef} position="relative" width="100%">
      <Input
        placeholder={placeholder}
        value={value || ""}
        onChange={handleInputChange}
        onFocus={handleFocus}
        onKeyDown={(e) => { if (e.key === "Escape") setOpen(false); }}
        size="sm"
        width="100%"
        {...inputProps}
      />
      {open && filteredItems.length > 0 && (
        <Box
          position="absolute"
          top="100%"
          left={0}
          right={0}
          zIndex={2000}
          background="white"
          border="1px solid"
          borderColor="gray.200"
          borderRadius="md"
          boxShadow="md"
          maxHeight="200px"
          overflowY="auto"
          _dark={{ background: "gray.800", borderColor: "gray.600" }}
        >
          {filteredItems.map((item) => (
            <Box
              key={item}
              onMouseDown={(e) => { e.preventDefault(); selectItem(item); }}
              _hover={{ bg: "gray.100", _dark: { bg: "gray.700" } }}
              padding="6px 12px"
              cursor="pointer"
              fontSize="sm"
            >
              {item}
            </Box>
          ))}
        </Box>
      )}
    </Box>
  );
};
