import React from "react";
import {
  Dialog,
  DialogBody,
  DialogCloseTrigger,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogRoot,
  DialogTitle,
  DialogBackdrop,
  DialogPositioner,
  Portal,
} from "@chakra-ui/react";
import { Button } from "../ui/button";

interface FormModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
  size?:
    | "xs"
    | "sm"
    | "md"
    | "lg"
    | "xl"
    | "2xl"
    | "3xl"
    | "4xl"
    | "5xl"
    | "6xl"
    | "full"
    | "default"
    | "large";
}

export const FormModal: React.FC<FormModalProps> = ({
  isOpen,
  onClose,
  title,
  children,
  footer,
  size = "default",
}) => {
  // Map size "default" to "md" and "large" to "xl" or "2xl"
  // Chakra default sizes are xs, sm, md, lg, xl, 2xl, etc.
  let chakraSize = size;
  if (size === "default") chakraSize = "md";
  if (size === "large") chakraSize = "xl";

  // Identify if the requested size is supported by the DialogRoot component directly
  // definition: "xs" | "sm" | "md" | "lg" | "xl" | "full" | "cover" | undefined
  const validRootSizes = ["xs", "sm", "md", "lg", "xl", "full", "cover"];
  const isRootSize = validRootSizes.includes(chakraSize);
  const rootSize = isRootSize ? (chakraSize as any) : undefined;
  const contentMaxW = !isRootSize ? chakraSize : undefined;

  return (
    <DialogRoot
      open={isOpen}
      onOpenChange={(e: { open: boolean }) => !e.open && onClose()}
      size={rootSize}
      scrollBehavior="inside"
    >
      <Portal>
        <DialogBackdrop />
        <DialogPositioner>
          <DialogContent maxW={contentMaxW}>
            <DialogHeader>
              <DialogTitle>{title}</DialogTitle>
            </DialogHeader>
            <DialogCloseTrigger />

            <DialogBody>{children}</DialogBody>

            {footer && <DialogFooter>{footer}</DialogFooter>}
          </DialogContent>
        </DialogPositioner>
      </Portal>
    </DialogRoot>
  );
};
