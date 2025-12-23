import { Progress } from "@chakra-ui/react"
import * as React from "react"

export interface ProgressBarProps extends React.ComponentProps<typeof Progress.Track> {
  label?: React.ReactNode
}

export const ProgressBar = React.forwardRef<HTMLDivElement, ProgressBarProps>(
  function ProgressBar(props, ref) {
    return (
      // @ts-ignore
      <Progress.Track {...props} ref={ref}>
        <Progress.Range />
      </Progress.Track>
    )
  },
)

export const ProgressRoot = React.forwardRef<HTMLDivElement, Progress.RootProps>(
  function ProgressRoot(props, ref) {
    return <Progress.Root {...props} ref={ref} />
  },
)

export const ProgressLabel = React.forwardRef<HTMLDivElement, Progress.LabelProps>(
  function ProgressLabel(props, ref) {
    return <Progress.Label {...props} ref={ref} />
  },
)

export const ProgressValueText = React.forwardRef<
  HTMLDivElement,
  Progress.ValueTextProps
>(function ProgressValueText(props, ref) {
  return <Progress.ValueText {...props} ref={ref} />
})
