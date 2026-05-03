import React from "react";
import { Box, Flex, Text } from "@chakra-ui/react";

interface ToggleFieldProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: React.ReactNode;
  hint?: string;
  disabled?: boolean;
}

export const ToggleField: React.FC<ToggleFieldProps> = ({
  checked,
  onChange,
  label,
  hint,
  disabled = false,
}) => {
  return (
    <Flex
      as="label"
      align="flex-start"
      gap={2.5}
      cursor={disabled ? "not-allowed" : "pointer"}
      userSelect="none"
      opacity={disabled ? 0.5 : 1}
    >
      {/* Track */}
      <Box
        display="inline-flex"
        w="32px"
        h="18px"
        borderRadius="full"
        bg={checked ? "blue.600" : "gray.300"}
        _dark={{ bg: checked ? "blue.500" : "gray.600" }}
        position="relative"
        transition="background 120ms linear"
        flexShrink={0}
        mt="2px"
        onClick={(e) => {
          e.preventDefault();
          if (!disabled) onChange(!checked);
        }}
      >
        {/* Thumb */}
        <Box
          position="absolute"
          top="2px"
          left={checked ? "16px" : "2px"}
          w="14px"
          h="14px"
          bg="white"
          borderRadius="full"
          transition="left 120ms linear"
          boxShadow="sm"
        />
      </Box>
      <Box display="flex" flexDirection="column" gap={0.5}>
        <Text fontSize="sm" color="fg">
          {label}
        </Text>
        {hint && (
          <Text fontSize="xs" color="fg.subtle">
            {hint}
          </Text>
        )}
      </Box>
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => !disabled && onChange(e.target.checked)}
        style={{ display: "none" }}
      />
    </Flex>
  );
};
