import React, { useCallback, useEffect, useRef, useState } from "react";
import { Box, Flex, Text, Portal } from "@chakra-ui/react";
import {
  DialogBackdrop,
  DialogContent,
  DialogPositioner,
  DialogRoot,
} from "@chakra-ui/react";
import { Button } from "../ui/button";
import { IconType } from "react-icons";
import {
  FiAlertCircle,
  FiCheck,
  FiLoader,
  FiX,
} from "react-icons/fi";

// --- Section definition ---
export interface SectionDef {
  id: string;
  label: string;
  icon: React.ReactElement | IconType;
}

// --- TwoPaneModal props ---
interface TwoPaneModalProps {
  isOpen: boolean;
  onClose: () => void;

  // Header
  title: string;
  subtitle?: string;
  headerIcon?: React.ReactElement;
  headerExtra?: React.ReactNode;

  // Sections & nav
  sections: SectionDef[];
  children: React.ReactNode;

  // Save bar
  dirty?: boolean;
  dirtyCount?: number;
  errorCount?: number;
  onSave?: () => void;
  onDiscard?: () => void;
  saving?: boolean;
  saveDisabled?: boolean;

  // Footer override (if you don't want the default save bar)
  footer?: React.ReactNode;

  width?: string | number;
}

export const TwoPaneModal: React.FC<TwoPaneModalProps> = ({
  isOpen,
  onClose,
  title,
  subtitle,
  headerIcon,
  headerExtra,
  sections,
  children,
  dirty = false,
  dirtyCount = 0,
  errorCount = 0,
  onSave,
  onDiscard,
  saving = false,
  saveDisabled = false,
  footer,
  width = "900px",
}) => {
  const scrollRef = useRef<HTMLDivElement>(null);
  const sectionRefs = useRef<Record<string, HTMLElement | null>>({});
  const [activeSection, setActiveSection] = useState(sections[0]?.id || "");

  // Scroll-spy
  useEffect(() => {
    const root = scrollRef.current;
    if (!root) return;
    const onScroll = () => {
      const top = root.scrollTop;
      let current = sections[0]?.id || "";
      for (const s of sections) {
        const el = sectionRefs.current[s.id];
        if (!el) continue;
        if (el.offsetTop - 80 <= top) current = s.id;
      }
      setActiveSection(current);
    };
    root.addEventListener("scroll", onScroll, { passive: true });
    onScroll();
    return () => root.removeEventListener("scroll", onScroll);
  }, [sections]);

  const scrollTo = useCallback(
    (id: string) => {
      const el = sectionRefs.current[id];
      const root = scrollRef.current;
      if (!el || !root) return;
      root.scrollTo({ top: el.offsetTop - 56, behavior: "smooth" });
      setActiveSection(id);
    },
    [],
  );

  // Provide ref-registration function to children via context
  const registerRef = useCallback((id: string, el: HTMLElement | null) => {
    sectionRefs.current[id] = el;
  }, []);

  return (
    <DialogRoot
      open={isOpen}
      onOpenChange={(e: { open: boolean }) => !e.open && onClose()}
      closeOnInteractOutside={false}
      size="xl"
    >
      <Portal>
        <DialogBackdrop />
        <DialogPositioner>
          <DialogContent
            maxW={width}
            height="86vh"
            maxH="820px"
            p={0}
            overflow="hidden"
            display="flex"
            flexDirection="column"
          >
            {/* Header */}
            <Flex
              align="center"
              gap={3}
              px={5}
              py={3.5}
              borderBottomWidth="1px"
              borderColor="border"
              flexShrink={0}
            >
              {headerIcon && (
                <Flex
                  w={7}
                  h={7}
                  borderRadius="sm"
                  bg="bg.subtle"
                  borderWidth="1px"
                  borderColor="border"
                  align="center"
                  justify="center"
                  color="fg.muted"
                  flexShrink={0}
                >
                  {headerIcon}
                </Flex>
              )}
              <Box flex={1} minW={0}>
                <Flex align="center" gap={2} flexWrap="nowrap" minW={0}>
                  <Text
                    fontSize="md"
                    fontWeight="semibold"
                    color="fg"
                    whiteSpace="nowrap"
                    flexShrink={0}
                  >
                    {title}
                  </Text>
                  {headerExtra}
                </Flex>
                {subtitle && (
                  <Text
                    fontSize="xs"
                    color="fg.subtle"
                    fontFamily="mono"
                    truncate
                  >
                    {subtitle}
                  </Text>
                )}
              </Box>
              <Box
                as="button"
                onClick={onClose}
                bg="transparent"
                border={0}
                cursor="pointer"
                color="fg.subtle"
                p={1.5}
                borderRadius="sm"
                flexShrink={0}
                _hover={{ bg: "bg.muted" }}
              >
                <FiX size={16} />
              </Box>
            </Flex>

            {/* Two-pane body */}
            <Flex flex={1} minH={0}>
              {/* Nav rail */}
              <Box
                as="nav"
                w="196px"
                borderRightWidth="1px"
                borderColor="border"
                py={3}
                px={2}
                flexShrink={0}
                bg="bg.subtle"
                overflowY="auto"
              >
                {sections.map((s) => {
                  const isActive = activeSection === s.id;
                  return (
                    <Box
                      key={s.id}
                      as="button"
                      onClick={() => scrollTo(s.id)}
                      display="flex"
                      alignItems="center"
                      gap={2}
                      w="full"
                      py={1.5}
                      px={2.5}
                      bg={isActive ? "bg.muted" : "transparent"}
                      border={0}
                      borderRadius="sm"
                      fontSize="sm"
                      fontWeight={isActive ? "medium" : "normal"}
                      color="fg"
                      cursor="pointer"
                      textAlign="left"
                      mb={0.5}
                      _hover={{ bg: isActive ? "bg.muted" : "bg.emphasized" }}
                    >
                      <Box
                        color={isActive ? "fg" : "fg.muted"}
                        display="inline-flex"
                      >
                        {React.isValidElement(s.icon)
                          ? s.icon
                          : React.createElement(s.icon as IconType, {
                              size: 14,
                            })}
                      </Box>
                      <Text flex={1}>{s.label}</Text>
                    </Box>
                  );
                })}
              </Box>

              {/* Scrolling content */}
              <Box
                ref={scrollRef}
                flex={1}
                overflowY="auto"
                bg="bg.subtle"
                p={5}
              >
                <TwoPaneContext.Provider value={{ registerRef }}>
                  {children}
                </TwoPaneContext.Provider>
                <Box h={10} />
              </Box>
            </Flex>

            {/* Footer / save bar */}
            {footer ? (
              <Box
                borderTopWidth="1px"
                borderColor="border"
                px={4}
                py={2.5}
                flexShrink={0}
              >
                {footer}
              </Box>
            ) : (
              <Flex
                align="center"
                gap={2.5}
                px={4}
                py={2.5}
                borderTopWidth="1px"
                borderColor="border"
                bg={dirty ? "orange.50" : "bg.panel"}
                _dark={{ bg: dirty ? "orange.950" : "bg.panel" }}
                transition="background 120ms linear"
                flexShrink={0}
              >
                <Flex align="center" gap={2} flex={1} minW={0}>
                  {dirty ? (
                    <>
                      <Box
                        w={2}
                        h={2}
                        borderRadius="full"
                        bg="orange.400"
                        display="inline-block"
                      />
                      <Text fontSize="sm" color="orange.800" _dark={{ color: "orange.200" }} fontWeight="medium">
                        {dirtyCount} unsaved{" "}
                        {dirtyCount === 1 ? "change" : "changes"}
                      </Text>
                      {errorCount > 0 && (
                        <Flex
                          align="center"
                          gap={1}
                          ml={2}
                          fontSize="xs"
                          color="red.600"
                          _dark={{ color: "red.300" }}
                        >
                          <FiAlertCircle size={12} />
                          {errorCount} {errorCount === 1 ? "error" : "errors"} to
                          fix
                        </Flex>
                      )}
                    </>
                  ) : (
                    <>
                      <Box color="green.500">
                        <FiCheck size={13} />
                      </Box>
                      <Text fontSize="sm" color="fg.muted">
                        All changes saved
                      </Text>
                    </>
                  )}
                </Flex>
                <Button variant="ghost" size="sm" onClick={onClose}>
                  Cancel
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={onDiscard}
                  disabled={!dirty}
                  opacity={dirty ? 1 : 0.45}
                >
                  Discard changes
                </Button>
                <Button
                  size="sm"
                  onClick={onSave}
                  disabled={!dirty || saveDisabled || errorCount > 0}
                  opacity={!dirty || saveDisabled || errorCount > 0 ? 0.5 : 1}
                >
                  {saving ? (
                    <>
                      <FiLoader size={12} className="spin" />
                      Saving…
                    </>
                  ) : (
                    <>
                      <FiCheck size={12} />
                      Save changes
                    </>
                  )}
                </Button>
              </Flex>
            )}
          </DialogContent>
        </DialogPositioner>
      </Portal>
    </DialogRoot>
  );
};

// --- Context for child sections to register refs ---
interface TwoPaneContextValue {
  registerRef: (id: string, el: HTMLElement | null) => void;
}

const TwoPaneContext = React.createContext<TwoPaneContextValue>({
  registerRef: () => {},
});

export const useTwoPaneRef = () => React.useContext(TwoPaneContext);

// --- TwoPaneSection: wraps each section's content with ref registration ---
interface TwoPaneSectionProps {
  id: string;
  children: React.ReactNode;
}

export const TwoPaneSection: React.FC<TwoPaneSectionProps> = ({
  id,
  children,
}) => {
  const { registerRef } = useTwoPaneRef();
  const ref = useCallback(
    (el: HTMLDivElement | null) => {
      registerRef(id, el);
    },
    [id, registerRef],
  );

  return (
    <Box ref={ref} data-section={id} scrollMarginTop="16px">
      {children}
    </Box>
  );
};
