import { Accordion, HStack } from "@chakra-ui/react";
import * as React from "react";
import { FiChevronDown } from "react-icons/fi";

interface AccordionItemTriggerProps extends Accordion.ItemTriggerProps {
  indicatorPlacement?: "start" | "end";
  children?: React.ReactNode;
}

export const AccordionItemTrigger = React.forwardRef<HTMLButtonElement, any>(
  function AccordionItemTrigger(props, ref) {
    const { children, indicatorPlacement = "end", ...rest } = props;
    return (
      // @ts-ignore
      <Accordion.ItemTrigger {...rest} ref={ref}>
        {indicatorPlacement === "start" && (
          // @ts-ignore
          <Accordion.ItemIndicator
            // @ts-ignore
            rotate={{ base: "-90deg", _open: "0deg" }}
          >
            <FiChevronDown />
          </Accordion.ItemIndicator>
        )}
        <HStack gap="4" flex="1" textAlign="start" width="full">
          {children}
        </HStack>
        {indicatorPlacement === "end" && (
          // @ts-ignore
          <Accordion.ItemIndicator>
            <FiChevronDown />
          </Accordion.ItemIndicator>
        )}
      </Accordion.ItemTrigger>
    );
  },
);

interface AccordionItemContentProps extends Accordion.ItemContentProps {
  children?: React.ReactNode;
}

export const AccordionItemContent = React.forwardRef<HTMLDivElement, any>(
  function AccordionItemContent(props, ref) {
    return (
      // @ts-ignore
      <Accordion.ItemContent>
        {/* @ts-ignore */}
        <Accordion.ItemBody {...props} ref={ref} />
      </Accordion.ItemContent>
    );
  },
);

export const AccordionRoot = React.forwardRef<HTMLDivElement, any>(
  function AccordionRoot(props, ref) {
    return (
      // @ts-ignore
      <Accordion.Root {...props} ref={ref} />
    );
  },
);

export const AccordionItem = React.forwardRef<HTMLDivElement, any>(
  function AccordionItem(props, ref) {
    return (
      // @ts-ignore
      <Accordion.Item {...props} ref={ref} />
    );
  },
);
