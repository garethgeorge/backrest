import { Box, Container } from "@chakra-ui/react";
import { BreadcrumbRoot, BreadcrumbLink, BreadcrumbCurrentLink } from "../components/ui/breadcrumb";
import React from "react";

interface BreadcrumbItem {
  title: string;
  onClick?: () => void;
}

export const MainContentAreaTemplate = ({
  breadcrumbs,
  children,
}: {
  breadcrumbs: BreadcrumbItem[];
  children: React.ReactNode;
}) => {
  return (
    <Box px={6} pb={6}>
      <BreadcrumbRoot my={4}>
        {breadcrumbs.map((b, i) => {
            const isLast = i === breadcrumbs.length - 1;
            if (isLast) {
                return <BreadcrumbCurrentLink key={i}>{b.title}</BreadcrumbCurrentLink>
            }
            return (
                 <BreadcrumbLink 
                    key={i} 
                    onClick={b.onClick} 
                    cursor={b.onClick ? "pointer" : "default"}
                    color={b.onClick ? "blue.500" : "inherit"}
                 >
                    {b.title}
                </BreadcrumbLink>
            )
        })}
      </BreadcrumbRoot>
      <Box
        p={6}
        m={0}
        minH={280}
        bg="bg.panel" // Using semantic token for generic background
        borderRadius="md"
        boxShadow="none" // Antd usually has no shadow on content bg, but Chakra Cards do. Keeping simple for now.
      >
        {children}
      </Box>
    </Box>
  );
};
