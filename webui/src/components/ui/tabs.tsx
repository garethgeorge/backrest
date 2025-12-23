import { Tabs as ChakraTabs } from "@chakra-ui/react"
import * as React from "react"

export const Tabs = ChakraTabs

export const TabsList = React.forwardRef<HTMLDivElement, any>(
    function TabsList(props, ref) {
        // @ts-ignore
        return <ChakraTabs.List ref={ref} {...props} />
    }
)

export const TabsTrigger = React.forwardRef<HTMLButtonElement, any>(
    function TabsTrigger(props, ref) {
        // @ts-ignore
        return <ChakraTabs.Trigger ref={ref} {...props} />
    }
)

export const TabsContent = React.forwardRef<HTMLDivElement, any>(
    function TabsContent(props, ref) {
        // @ts-ignore
        return <ChakraTabs.Content ref={ref} {...props} />
    }
)

export const TabsRoot = React.forwardRef<HTMLDivElement, any>(
    function TabsRoot(props, ref) {
        // @ts-ignore
        return <ChakraTabs.Root ref={ref} {...props} />
    }
)

export const TabsIndicator = ChakraTabs.Indicator
