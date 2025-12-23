
import type { ButtonProps as ChakraButtonProps } from "@chakra-ui/react"
import {
  AbsoluteCenter,
  Button as ChakraButton,
  Span,
  Spinner,
} from "@chakra-ui/react"
import * as React from "react"

interface ButtonLoadingProps {
  loading?: boolean
  loadingText?: React.ReactNode
}

export interface ButtonProps extends ChakraButtonProps, ButtonLoadingProps {}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  function Button(props, ref) {
    const { loading, disabled, loadingText, children, ...rest } = props
    return (
      <ChakraButton disabled={loading || disabled} ref={ref} {...rest}>
        {loading && !loadingText ? (
          <AbsoluteCenter display="inline-flex">
            <Spinner size="inherit" color="inherit" />
          </AbsoluteCenter>
        ) : (
          <ButtonLoadingOverlay loading={loading} loadingText={loadingText} />
        )}
        {loading ? (
          <Span opacity={0}>{children}</Span>
        ) : (
          children
        )}
      </ChakraButton>
    )
  },
)

function ButtonLoadingOverlay(props: ButtonLoadingProps) {
  const { loading, loadingText } = props
  if (!loading || !loadingText) return null
  return (
    <AbsoluteCenter display="inline-flex" alignItems="center" gap="2">
      <Spinner size="inherit" color="inherit" />
      {loadingText}
    </AbsoluteCenter>
  )
}
