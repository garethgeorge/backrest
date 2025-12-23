import React, { useEffect, useState, useMemo } from "react";
import { createListCollection } from "@chakra-ui/react";
import {
  ComboboxRoot,
  ComboboxInput,
  ComboboxContent,
  ComboboxItem,
  ComboboxControl,
  ComboboxEmpty
} from "./ui/combobox";
import { debounce } from "../lib/util";
import { backrestService } from "../api";
import { isWindows } from "../state/buildcfg";
import { StringList } from "../../gen/ts/types/value_pb";

const sep = isWindows ? "\\" : "/";

export const URIAutocomplete = (props: any) => {
  const { value, onChange, placeholder, disabled } = props;
  // value is string
  const [items, setItems] = useState<{ label: string; value: string }[]>([]);
  
  // Create collection for Combobox
  const collection = createListCollection({ items });

  // Debounced fetch
  const fetchOptions = useMemo(
    () =>
      debounce((inputVal: string) => {
        // Only fetch if input has path separators or is long enough?
        // Logic from original: 
        const lastSlash = inputVal.lastIndexOf(sep);
        let searchVal = inputVal;
        if (lastSlash !== -1) {
            searchVal = inputVal.substring(0, lastSlash);
        } else if (searchVal === "") {
             return; // Don't fetch root on empty?
        }

        backrestService
          .pathAutocomplete({ value: searchVal + sep })
          .then((res: StringList) => {
            if (!res.values) return;
            const newItems = res.values.map((v) => ({
                label: searchVal + sep + v,
                value: searchVal + sep + v
            }));
            setItems(newItems);
          })
          .catch((e) => console.error("Path autocomplete error:", e));
      }, 300),
    []
  );

  const handleInputChange = (e: any) => {
      const val = e.target.value;
      if (onChange) onChange(val);
      fetchOptions(val);
  };

  const handleOpenChange = (e: any) => {
      if (e.open) {
          fetchOptions(value || "");
      }
  }

  return (
      <ComboboxRoot
        collection={collection}
        inputBehavior="autocomplete" // allows nice typing
        disabled={disabled}
        inputValue={value}
        onInputValueChange={(e: any) => {
             // Combobox updates inputValue when typing
             if (onChange) onChange(e.inputValue);
             fetchOptions(e.inputValue);
        }}
        onOpenChange={handleOpenChange}
      >
          <ComboboxControl>
              <ComboboxInput placeholder={placeholder} />
          </ComboboxControl>
          <ComboboxContent>
              {items.length === 0 && <ComboboxEmpty>No paths found</ComboboxEmpty>}
              {items.map(item => (
                  <ComboboxItem key={item.value} item={item}>
                      {item.label}
                  </ComboboxItem>
              ))}
          </ComboboxContent>
      </ComboboxRoot>
  );
};
