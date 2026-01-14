import {
  Pagination as ChakraPagination,
  IconButton,
  Text,
  Flex,
  Button,
} from "@chakra-ui/react";
import { FiChevronLeft, FiChevronRight } from "react-icons/fi";
import * as React from "react";

export interface PaginationRootProps extends ChakraPagination.RootProps {}

export const PaginationRoot = React.forwardRef<
  HTMLDivElement,
  PaginationRootProps
>(function PaginationRoot(props, ref) {
  return (
    <ChakraPagination.Root {...props} ref={ref}>
      <Flex gap={2} alignItems="center">
        {props.children}
      </Flex>
    </ChakraPagination.Root>
  );
});

export const PaginationPrevTrigger = React.forwardRef<
  HTMLButtonElement,
  ChakraPagination.PrevTriggerProps
>(function PaginationPrevTrigger(props, ref) {
  return (
    // @ts-ignore
    <ChakraPagination.PrevTrigger {...props} asChild ref={ref}>
      <IconButton variant="ghost" aria-label="Previous Page" size="sm">
        <FiChevronLeft />
      </IconButton>
    </ChakraPagination.PrevTrigger>
  );
});

export const PaginationNextTrigger = React.forwardRef<
  HTMLButtonElement,
  ChakraPagination.NextTriggerProps
>(function PaginationNextTrigger(props, ref) {
  return (
    // @ts-ignore
    <ChakraPagination.NextTrigger {...props} asChild ref={ref}>
      <IconButton variant="ghost" aria-label="Next Page" size="sm">
        <FiChevronRight />
      </IconButton>
    </ChakraPagination.NextTrigger>
  );
});

export const PaginationItems = (props: any) => {
  return (
    <ChakraPagination.Items
      {...props}
      // @ts-ignore
      render={(page: any) => (
        // @ts-ignore
        <ChakraPagination.Item
          key={page.value}
          value={page.value}
          asChild
          type={page.type}
        >
          <Button
            variant={page.type === "page" ? "ghost" : "plain"}
            size="sm"
            px={2}
            minW={8}
          >
            {page.value}
          </Button>
        </ChakraPagination.Item>
      )}
    />
  );
};

export const PaginationPageText = React.forwardRef<
  HTMLParagraphElement,
  ChakraPagination.PageTextProps
>(function PaginationPageText(props, ref) {
  return (
    <ChakraPagination.PageText {...props} ref={ref} asChild>
      <Text fontSize="sm" />
    </ChakraPagination.PageText>
  );
});
