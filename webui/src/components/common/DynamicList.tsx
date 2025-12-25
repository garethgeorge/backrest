import {
  Flex,
  Stack,
  IconButton,
  Box,
  HStack,
  Text as CText,
} from "@chakra-ui/react";
import React, { useEffect, useState } from "react";
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

const uid = () => Math.random().toString(36).slice(2, 9);

export const DynamicList = ({
  label,
  items,
  onUpdate,
  tooltip,
  placeholder,
  required,
}: any) => {
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
                      <URIAutocomplete
                        id={`${label}-${index}`}
                        value={item}
                        onChange={(val: string) => handleChange(index, val)}
                        placeholder={placeholder}
                      />
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
        <CText fontSize="xs" color="fg.muted">
          {tooltip}
        </CText>
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
