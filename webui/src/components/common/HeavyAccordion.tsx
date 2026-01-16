import React, { useEffect, useState } from "react";
import {
  AccordionItem,
  AccordionItemContent,
  AccordionItemTrigger,
  AccordionRoot,
} from "../ui/accordion";
import { Text } from "@chakra-ui/react";

export interface HeavyAccordionItem {
  key: string;
  label: string;
  children: React.ReactNode;
}

interface HeavyAccordionProps {
  items: HeavyAccordionItem[];
  defaultExpanded?: string[];
}

export const HeavyAccordion = ({
  items,
  defaultExpanded = [],
}: HeavyAccordionProps) => {
  const [visitedKeys, setVisitedKeys] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (defaultExpanded.length > 0) {
      setVisitedKeys((prev) => {
        let changed = false;
        for (const k of defaultExpanded) {
          if (!prev.has(k)) {
            changed = true;
            break;
          }
        }
        if (changed) {
          const next = new Set(prev);
          defaultExpanded.forEach((k) => next.add(k));
          return next;
        }
        return prev;
      });
    }
  }, [defaultExpanded.join(",")]);

  const handleAccordionChange = (e: { value: string[] }) => {
    setVisitedKeys((prev) => {
      let changed = false;
      for (const k of e.value) {
        if (!prev.has(k)) {
          changed = true;
          break;
        }
      }
      if (changed) {
        const next = new Set(prev);
        e.value.forEach((k) => next.add(k));
        return next;
      }
      return prev;
    });
  };

  return (
    <AccordionRoot
      collapsible
      multiple
      defaultValue={defaultExpanded}
      variant="plain"
      onValueChange={handleAccordionChange}
    >
      {items.map((item) => (
        <AccordionItem key={item.key} value={item.key} border="none">
          <AccordionItemTrigger py={2}>
            <Text fontSize="sm" fontWeight="medium">
              {item.label}
            </Text>
          </AccordionItemTrigger>
          <AccordionItemContent pb={4}>
            {visitedKeys.has(item.key) || defaultExpanded.includes(item.key)
              ? item.children
              : null}
          </AccordionItemContent>
        </AccordionItem>
      ))}
    </AccordionRoot>
  );
};
