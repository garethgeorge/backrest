import React from "react";
import { Box, Flex, Text } from "@chakra-ui/react";
import { IconType } from "react-icons";

interface SectionCardProps {
  id?: string;
  icon?: React.ReactElement | IconType;
  title: string;
  description?: string;
  children: React.ReactNode;
  cardRef?: React.Ref<HTMLDivElement>;
}

export const SectionCard: React.FC<SectionCardProps> = ({
  id,
  icon,
  title,
  description,
  children,
  cardRef,
}) => {
  return (
    <Box
      ref={cardRef}
      data-section={id}
      bg="bg.panel"
      borderWidth="1px"
      borderColor="border"
      borderRadius="md"
      mb={4}
      scrollMarginTop="16px"
    >
      <Flex
        align="center"
        gap={2.5}
        px={5}
        py={3.5}
        borderBottomWidth="1px"
        borderColor="border"
      >
        {icon && (
          <Box color="fg.muted" fontSize="md">
            {React.isValidElement(icon)
              ? icon
              : React.createElement(icon as IconType, { size: 16 })}
          </Box>
        )}
        <Box>
          <Text fontSize="sm" fontWeight="semibold" color="fg">
            {title}
          </Text>
          {description && (
            <Text fontSize="xs" color="fg.subtle" mt={0.5}>
              {description}
            </Text>
          )}
        </Box>
      </Flex>
      <Box p={5}>{children}</Box>
    </Box>
  );
};
