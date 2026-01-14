import { NumberInput, HStack, IconButton } from "@chakra-ui/react";
import { LuMinus, LuPlus } from "react-icons/lu";
import React from "react";
import { Field } from "../ui/field";

interface NumberInputProps extends NumberInput.RootProps {
  label?: string;
  helperText?: string;
  errorText?: string;
}

export const NumberInputField = React.forwardRef<
  HTMLDivElement,
  NumberInputProps
>(function NumberInputField(props, ref) {
  const { label, helperText, errorText, ...rest } = props;
  return (
    <Field label={label} helperText={helperText} errorText={errorText}>
      <NumberInput.Root ref={ref} {...rest} width="full">
        <HStack gap="2" width="full">
          <NumberInput.Scrubber />
          {/* @ts-ignore */}
          <NumberInput.Input style={{ flex: 1 }} />
        </HStack>
      </NumberInput.Root>
    </Field>
  );
});
