import { ProgressCircle as ChakraProgressCircle } from "@chakra-ui/react"
import * as React from "react"

export interface ProgressCircleProps extends ChakraProgressCircle.RootProps {
  trackColor?: string
  rangeColor?: string
}

export const ProgressCircle = React.forwardRef<
  HTMLDivElement,
  ProgressCircleProps
>(function ProgressCircle(props, ref) {
  const { trackColor, rangeColor, children, ...rest } = props
  return (
    <ChakraProgressCircle.Root {...rest} ref={ref}>
      {/* @ts-ignore */}
      <ChakraProgressCircle.Circle stroke={trackColor}>
        {/* @ts-ignore */}
        <ChakraProgressCircle.Track stroke={rangeColor} />
      </ChakraProgressCircle.Circle>
      {children}
    </ChakraProgressCircle.Root>
  )
})

export const ProgressCircleRoot = ChakraProgressCircle.Root
export const ProgressCircleRing = ChakraProgressCircle.Circle
export const ProgressCircleTrack = ChakraProgressCircle.Track
export const ProgressCircleRange = ChakraProgressCircle.Range
export const ProgressCircleValueText = ChakraProgressCircle.ValueText
