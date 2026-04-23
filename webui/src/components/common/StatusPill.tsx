import React from "react";
import { Box } from "@chakra-ui/react";

type PillTone = "neutral" | "ok" | "warn" | "error" | "info" | "mono";

interface StatusPillProps {
  tone?: PillTone;
  children: React.ReactNode;
}

const toneStyles: Record<PillTone, { bg: string; color: string }> = {
  neutral: { bg: "gray.100", color: "gray.700" },
  ok: { bg: "green.100", color: "green.700" },
  warn: { bg: "orange.100", color: "orange.700" },
  error: { bg: "red.100", color: "red.700" },
  info: { bg: "blue.100", color: "blue.700" },
  mono: { bg: "gray.100", color: "gray.900" },
};

export const StatusPill: React.FC<StatusPillProps> = ({
  tone = "neutral",
  children,
}) => {
  const styles = toneStyles[tone];
  return (
    <Box
      as="span"
      display="inline-flex"
      alignItems="center"
      gap={1}
      px={1.5}
      py={0.5}
      borderRadius="sm"
      fontSize="xs"
      fontWeight="medium"
      fontFamily={tone === "mono" ? "mono" : undefined}
      bg={styles.bg}
      color={styles.color}
      _dark={{
        bg: tone === "mono" ? "gray.800" : undefined,
        color: tone === "mono" ? "gray.100" : undefined,
      }}
    >
      {children}
    </Box>
  );
};
