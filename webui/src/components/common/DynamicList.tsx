import {
  Flex,
  Stack,
  IconButton,
  Box,
  HStack,
  Text as CText,
  Input,
  createListCollection,
} from "@chakra-ui/react";
import React, { useEffect, useState, useMemo } from "react";
import { FiPlus as Plus, FiMinus as Minus, FiMenu } from "react-icons/fi";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Button } from "../ui/button";
import { URIAutocomplete } from "./URIAutocomplete";
import {
  ComboboxRoot,
  ComboboxInput,
  ComboboxContent,
  ComboboxItem,
  ComboboxControl,
  ComboboxEmpty,
} from "../ui/combobox";

interface DynamicListProps {
  label: string;
  items: string[];
  onUpdate: (items: string[]) => void;
  tooltip?: React.ReactNode;
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
  id,
}: {
  value: string;
  onChange: (val: string) => void;
  placeholder?: string;
  id?: string;
}) => {
  const [inputVal, setInputVal] = useState(value);

  useEffect(() => {
    setInputVal(value);
  }, [value]);

  const collection = useMemo(() => {
    return createListCollection({
      items: RESTIC_FLAGS.filter((item) =>
        item.value.startsWith(inputVal || ""),
      ),
    });
  }, [inputVal]);

  const handleInputChange = (e: any) => {
    const val = e.inputValue;
    setInputVal(val);
    onChange(val);
  };

  return (
    <ComboboxRoot
      id={id}
      collection={collection}
      inputBehavior="autocomplete"
      inputValue={inputVal}
      onInputValueChange={handleInputChange}
      onValueChange={(e: any) => {
        if (e.value && e.value[0]) {
          setInputVal(e.value[0]);
          onChange(e.value[0]);
        }
      }}
      allowCustomValue
      width="full"
    >
      <ComboboxControl hideTrigger>
        <ComboboxInput placeholder={placeholder} width="full" />
      </ComboboxControl>
      <ComboboxContent zIndex={2000}>
        <ComboboxEmpty>No flags found</ComboboxEmpty>
        {collection.items.map((item) => (
          <ComboboxItem key={item.value} item={item}>
            {item.label}
          </ComboboxItem>
        ))}
      </ComboboxContent>
    </ComboboxRoot>
  );
};

const uid = () => Math.random().toString(36).slice(2, 9);

export const DynamicList = ({
  label,
  items,
  onUpdate,
  tooltip,
  placeholder,
  required,
  autocompleteType = "none",
}: DynamicListProps) => {
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

  const [ids, setIds] = useState<string[]>(() => items.map(() => uid()));

  // Sync IDs if items length changes externally (e.g. reset/template load)
  useEffect(() => {
    while (ids.length < items.length) ids.push(uid());
    while (ids.length > items.length) ids.pop();
  }, [items]);

  const handleChange = (index: number, val: string) => {
    const newItems = [...items];
    newItems[index] = val;
    onUpdate(newItems);
  };

  const handleAdd = () => {
    onUpdate([...items, ""]);
    setIds([...ids, uid()]);
  };

  const handleRemove = (index: number) => {
    const newItems = [...items];
    newItems.splice(index, 1);
    onUpdate(newItems);

    const newIds = [...ids];
    newIds.splice(index, 1);
    setIds(newIds);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    if (over && active.id !== over.id) {
      const oldIndex = ids.indexOf(active.id as string);
      const newIndex = ids.indexOf(over.id as string);

      if (oldIndex !== -1 && newIndex !== -1) {
        onUpdate(arrayMove(items, oldIndex, newIndex));
        setIds(arrayMove(ids, oldIndex, newIndex));
      }
    }
  };

  return (
    <Stack gap={1.5} width="full">
      {label && (
        <CText fontSize="sm" fontWeight="medium">
          {label}{" "}
          {required && (
            <CText as="span" color="red.500">
              *
            </CText>
          )}
        </CText>
      )}
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
      >
        <Stack gap={2} width="full">
          <SortableContext items={ids} strategy={verticalListSortingStrategy}>
            {items.map((item: string, index: number) => {
              // Fallback if IDs are out of sync temporarily (should fix next render)
              const id = ids[index] || `temp-${index}`;
              return (
                <SortableItem key={id} id={id} index={index}>
                  <HStack gap={2} width="full">
                    <Box flex={1}>
                      {autocompleteType === "uri" ? (
                        <URIAutocomplete
                          id={`${label}-${index}`}
                          value={item}
                          onChange={(val: string) => handleChange(index, val)}
                          placeholder={placeholder}
                        />
                      ) : autocompleteType === "flag" ? (
                        <FlagAutocomplete
                          id={`${label}-${index}`}
                          value={item}
                          onChange={(val: string) => handleChange(index, val)}
                          placeholder={placeholder}
                        />
                      ) : (
                        <Input
                          id={`${label}-${index}`}
                          value={item}
                          onChange={(e) => handleChange(index, e.target.value)}
                          placeholder={placeholder}
                          size="sm"
                        />
                      )}
                    </Box>
                    <IconButton
                      size="sm"
                      variant="ghost"
                      onClick={() => handleRemove(index)}
                      aria-label="Remove"
                    >
                      <Minus size={16} />
                    </IconButton>
                  </HStack>
                </SortableItem>
              );
            })}
          </SortableContext>
          <Flex align="center" gap={2} width="full">
            {/* Placeholder for drag handle */}
            <Box width="16px" display="flex" justifyContent="center">
              <FiMenu style={{ opacity: 0 }} />
            </Box>
            <HStack gap={2} flex={1}>
              <Button
                size="sm"
                variant="outline"
                borderStyle="dashed"
                onClick={handleAdd}
                flex={1}
              >
                <Plus size={16} /> Add
              </Button>
              {/* Placeholder for remove button */}
              <Box width="32px" />
            </HStack>
          </Flex>
        </Stack>
      </DndContext>
      {tooltip && (
        <Box fontSize="xs" color="fg.muted">
          {tooltip}
        </Box>
      )}
    </Stack>
  );
};

const SortableItem = (props: any) => {
  const { attributes, listeners, setNodeRef, transform, transition } =
    useSortable({ id: props.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <div ref={setNodeRef} style={style} className="sortable-item">
      <Flex align="center" gap={2}>
        <div
          {...attributes}
          {...listeners}
          style={{ cursor: "grab", display: "flex", alignItems: "center" }}
        >
          <FiMenu color="gray" />
        </div>
        <Box flex="1">{props.children}</Box>
      </Flex>
    </div>
  );
};
