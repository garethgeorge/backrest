import { Splitter as ChakraSplitter } from "@chakra-ui/react"
import * as React from "react"

export interface SplitterProps extends ChakraSplitter.RootProps {
  children?: React.ReactNode
}

export const Splitter = React.forwardRef<HTMLDivElement, SplitterProps>(
  function Splitter(props, ref) {
    const { children, ...rest } = props
    return (
      <ChakraSplitter.Root ref={ref} {...(rest as any)}>
        {children}
      </ChakraSplitter.Root>
    )
  },
)

export interface SplitterPanelProps extends ChakraSplitter.PanelProps {
  children?: React.ReactNode
  id?: string
}

export const SplitterPanel = React.forwardRef<
  HTMLDivElement,
  SplitterPanelProps
>(function SplitterPanel(props, ref) {
  return <ChakraSplitter.Panel {...(props as any)} ref={ref} />
})

export interface SplitterResizeTriggerProps extends ChakraSplitter.ResizeTriggerProps {
    id?: string
}

export const SplitterResizeTrigger = React.forwardRef<
  HTMLButtonElement,
  SplitterResizeTriggerProps
>(function SplitterResizeTrigger(props, ref) {
  return <ChakraSplitter.ResizeTrigger {...(props as any)} ref={ref} />
})

export const SplitterRoot = ChakraSplitter.Root
export const SplitterPanelPart = ChakraSplitter.Panel
export const SplitterResizeTriggerPart = ChakraSplitter.ResizeTrigger
