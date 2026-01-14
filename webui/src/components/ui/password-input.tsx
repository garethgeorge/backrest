"use client";

import type { ButtonProps, GroupProps, InputProps } from "@chakra-ui/react";
import {
  Box,
  HStack,
  IconButton,
  Input,
  mergeRefs,
  useControllableState,
} from "@chakra-ui/react";
import * as React from "react";
import { LuEye, LuEyeOff } from "react-icons/lu";
import { InputGroup } from "./input-group";

export interface PasswordInputProps extends InputProps {
  rootProps?: GroupProps;
  attach?: React.ReactNode;
}

export const PasswordInput = React.forwardRef<
  HTMLInputElement,
  PasswordInputProps
>(function PasswordInput(props, ref) {
  const { rootProps, attach, ...rest } = props;
  const [visible, setVisible] = useControllableState({ defaultValue: false });
  const inputRef = React.useRef<HTMLInputElement>(null);

  return (
    <InputGroup
      width="full"
      endElement={
        <Box display="flex" gap="1" alignItems="center">
          {attach}
          <IconButton
            variant="ghost"
            aria-label={visible ? "Hide password" : "Show password"}
            onClick={() => {
              setVisible(!visible);
              inputRef.current?.focus();
            }}
          >
            {visible ? <LuEyeOff /> : <LuEye />}
          </IconButton>
        </Box>
      }
      {...rootProps}
    >
      <Input
        {...rest}
        ref={mergeRefs(ref, inputRef)}
        type={visible ? "text" : "password"}
      />
    </InputGroup>
  );
});
