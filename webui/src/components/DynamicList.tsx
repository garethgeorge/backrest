import {
  Flex,
  Stack,
  IconButton,
  Input,
  createListCollection,
} from "@chakra-ui/react";
import { FiPlus as Plus, FiMinus as Minus } from "react-icons/fi";
import { Button } from "./ui/button";
import { Field } from "./ui/field";
import { URIAutocomplete } from "./URIAutocomplete";
import {
  ComboboxRoot,
  ComboboxInput,
  ComboboxContent,
  ComboboxItem,
  ComboboxControl,
  ComboboxEmpty,
} from "./ui/combobox";
import React, { useMemo, useState } from "react";

interface DynamicListProps {
  label: string;
  items: string[];
  onUpdate: (items: string[]) => void;
  tooltip?: string;
  placeholder?: string;
  required?: boolean;
  autocompleteType?: "uri" | "flag" | "none";
}

const RESTIC_FLAGS = [
  { label: "--cacert <file>", value: "--cacert" },
  { label: "--cache-dir <directory>", value: "--cache-dir" },
  { label: "--cleanup-cache", value: "--cleanup-cache" },
  { label: "--compression <mode>", value: "--compression" },
  { label: "--http-user-agent <string>", value: "--http-user-agent" },
  { label: "--insecure-no-password", value: "--insecure-no-password" },
  { label: "--insecure-tls", value: "--insecure-tls" },
  { label: "--json", value: "--json" },
  { label: "--key-hint <key>", value: "--key-hint" },
  { label: "--limit-download <rate>", value: "--limit-download" },
  { label: "--limit-upload <rate>", value: "--limit-upload" },
  { label: "--no-cache", value: "--no-cache" },
  { label: "--no-extra-verify", value: "--no-extra-verify" },
  { label: "--no-lock", value: "--no-lock" },
  { label: "--option <key=value>", value: "--option" },
  { label: "--pack-size <size>", value: "--pack-size" },
  { label: "--password-command <command>", value: "--password-command" },
  { label: "--password-file <file>", value: "--password-file" },
  { label: "--quiet", value: "--quiet" },
  { label: "--repo <repository>", value: "--repo" },
  { label: "--repository-file <file>", value: "--repository-file" },
  { label: "--retry-lock <duration>", value: "--retry-lock" },
  { label: "--tls-client-cert <file>", value: "--tls-client-cert" },
  { label: "--verbose", value: "--verbose" },
];

const FlagAutocomplete = ({
  value,
  onChange,
  placeholder,
}: {
  value: string;
  onChange: (val: string) => void;
  placeholder?: string;
}) => {
  // Basic filtering based on input
  const [inputVal, setInputVal] = useState(value);

  // Sync internal state if external value changes (optional, but good for controlled components)
  React.useEffect(() => {
    setInputVal(value);
  }, [value]);

  const collection = useMemo(() => {
    return createListCollection({ items: RESTIC_FLAGS });
  }, []);

  const handleInputChange = (e: any) => {
    const val = e.inputValue;
    setInputVal(val);
    onChange(val);
  };

  return (
    <ComboboxRoot
      collection={collection}
      inputBehavior="autocomplete"
      inputValue={inputVal}
      onInputValueChange={handleInputChange}
    >
      <ComboboxControl>
        <ComboboxInput placeholder={placeholder} />
      </ComboboxControl>
      <ComboboxContent>
        <ComboboxEmpty>No flags found</ComboboxEmpty>
        {RESTIC_FLAGS.map((item) => (
          <ComboboxItem key={item.value} item={item}>
            {item.label}
          </ComboboxItem>
        ))}
      </ComboboxContent>
    </ComboboxRoot>
  );
};

export const DynamicList = ({
  label,
  items,
  onUpdate,
  tooltip,
  placeholder,
  required,
  autocompleteType = "none",
}: DynamicListProps) => {
  const handleChange = (index: number, val: string) => {
    const newItems = [...items];
    newItems[index] = val;
    onUpdate(newItems);
  };

  const handleAdd = () => {
    onUpdate([...items, ""]);
  };

  const handleRemove = (index: number) => {
    const newItems = [...items];
    newItems.splice(index, 1);
    onUpdate(newItems);
  };

  const renderInput = (item: string, index: number) => {
    if (autocompleteType === "uri") {
      return (
        <URIAutocomplete
          value={item}
          onChange={(val: string) => handleChange(index, val)}
          placeholder={placeholder}
        />
      );
    } else if (autocompleteType === "flag") {
      return (
        <FlagAutocomplete
          value={item}
          onChange={(val: string) => handleChange(index, val)}
          placeholder={placeholder}
        />
      );
    } else {
      return (
        <Input
          value={item}
          onChange={(e) => handleChange(index, e.target.value)}
          placeholder={placeholder}
        />
      );
    }
  };

  return (
    <Field label={label} helperText={tooltip} required={required}>
      <Stack gap={2}>
        {items.map((item: string, index: number) => (
          <Flex key={index} gap={2}>
            {renderInput(item, index)}
            <IconButton
              size="sm"
              variant="ghost"
              onClick={() => handleRemove(index)}
              aria-label="Remove"
            >
              <Minus size={16} />
            </IconButton>
          </Flex>
        ))}
        <Button size="sm" variant="outline" onClick={handleAdd}>
          <Plus size={16} /> Add
        </Button>
      </Stack>
    </Field>
  );
};
