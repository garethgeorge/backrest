"use client";

import {
  Toaster as ChakraToaster,
  Portal,
  Spinner,
  Stack,
  Toast,
  createToaster,
} from "@chakra-ui/react";

// Toasts sit above dialogs, so they must not cover dialog footers where the
// primary actions (Submit/Save) live; top-end keeps them clear of those.
export const toaster = createToaster({
  placement: "top-end",
  pauseOnPageIdle: true,
});

export const Toaster = () => {
  return (
    <Portal>
      <ChakraToaster toaster={toaster}>
        {(toast: any) => (
          <Toast.Root width={{ md: "500px" }}>
            {toast.type === "loading" ? (
              <Spinner size="sm" color="blue.solid" />
            ) : (
              // @ts-ignore
              <Toast.Indicator />
            )}
            <Stack gap="1" flex="1" maxWidth="100%">
              {/* @ts-ignore */}
              {toast.title && <Toast.Title>{toast.title}</Toast.Title>}
              {toast.description && (
                // @ts-ignore
                <Toast.Description>{toast.description}</Toast.Description>
              )}
            </Stack>
            {toast.action && (
              // @ts-ignore
              <Toast.ActionTrigger>{toast.action.label}</Toast.ActionTrigger>
            )}
            {/* @ts-ignore */}
            <Toast.CloseTrigger />
          </Toast.Root>
        )}
      </ChakraToaster>
    </Portal>
  );
};
