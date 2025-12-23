"use client"

import {
  Toaster as ChakraToaster,
  Portal,
  Spinner,
  Stack,
  Toast,
  createToaster,
} from "@chakra-ui/react"

export const toaster = createToaster({
  placement: "bottom-end",
  pauseOnPageIdle: true,
})

export const Toaster = () => {
  return (
    <Portal>
      {/* @ts-ignore */}
      <ChakraToaster toaster={toaster} inset="0">
        {(toast: any) => (
          <Toast.Root width={{ md: "sm" }}>
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
            {toast.meta?.closable && <Toast.CloseTrigger />}
          </Toast.Root>
        )}
      </ChakraToaster>
    </Portal>
  )
}
