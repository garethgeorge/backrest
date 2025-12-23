import React from "react";
import { Box, Container, Breadcrumb, Heading, Flex } from "@chakra-ui/react";
import { useColorModeValue } from "@/components/ui/color-mode";

interface BreadcrumbItem {
  title: React.ReactNode;
  href?: string;
}

interface PageTemplateProps {
    heading?: React.ReactNode;
    breadcrumbs?: BreadcrumbItem[];
    children: React.ReactNode;
    actions?: React.ReactNode;
}

export const PageTemplate: React.FC<PageTemplateProps> = ({
    heading,
    breadcrumbs,
    children,
    actions,
}) => {
    // Subtle background for layout distinction if needed, currently transparent
    const bg = useColorModeValue("white", "gray.900");

    return (
        <Flex direction="column" height="100%" width="100%">
             {/* Header Section */}
             {(breadcrumbs || heading || actions) && (
                <Box 
                    py={4} 
                    px={8} 
                    borderBottomWidth="1px" 
                    bg={bg}
                >
                    {/* Breadcrumbs */}
                    {breadcrumbs && breadcrumbs.length > 0 && (
                        <Breadcrumb.Root size="sm" mb={actions || heading ? 2 : 0}>
                            {breadcrumbs.map((item, index) => (
                                <Breadcrumb.Link key={index} href={item.href}>
                                    {item.title}
                                </Breadcrumb.Link>
                            ))}
                        </Breadcrumb.Root>
                    )}

                    <Flex justify="space-between" align="center">
                        {heading && (
                            <Heading size="xl" fontWeight="bold">
                                {heading}
                            </Heading>
                        )}
                        {actions && (
                            <Box>
                                {actions}
                            </Box>
                        )}
                    </Flex>
                </Box>
             )}

            {/* Main Content */}
            <Box flex="1" overflow="auto" p={8}>
                <Container maxW="6xl" p={0}>
                    {children}
                </Container>
            </Box>
        </Flex>
    );
};
