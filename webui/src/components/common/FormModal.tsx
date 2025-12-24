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
  size?: "default" | "large";
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
  const chakraSize = size === "large" ? "xl" : "md";

  return (
    <DialogRoot
      open={isOpen}
      onOpenChange={(e: { open: boolean }) => !e.open && onClose()}
      size={chakraSize}
      scrollBehavior="inside"
    >
      <Portal>
        {/* @ts-ignore */}
        <DialogBackdrop />
        {/* @ts-ignore */}
        <DialogPositioner>
          {/* @ts-ignore */}
          <DialogContent>
            <DialogHeader>
              {/* @ts-ignore */}
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
