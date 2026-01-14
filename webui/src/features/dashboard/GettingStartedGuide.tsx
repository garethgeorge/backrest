import {
  Heading,
  Text,
  Separator,
  Box,
  HStack,
  List,
  Spinner,
} from "@chakra-ui/react";
import {
  AccordionRoot,
  AccordionItem,
  AccordionItemTrigger,
  AccordionItemContent,
} from "../../components/ui/accordion";
import { Link } from "../../components/ui/link";
import React from "react";
import { useConfig } from "../../app/provider";
import { ConfigSchema } from "../../../gen/ts/v1/config_pb";
import { isDevBuild } from "../../state/buildcfg";
import { toJsonString } from "@bufbuild/protobuf";

export const GettingStartedGuide = () => {
  const [config] = useConfig();

  const DividerWithText = ({ children }: { children: React.ReactNode }) => (
    <HStack gap={4} width="full" my={4}>
      <Text fontWeight="bold" whiteSpace="nowrap" color="gray.500">
        {children}
      </Text>
      <Separator flex="1" />
    </HStack>
  );

  return (
    <Box>
      <Heading size="xl" mb={4}>
        Getting Started
      </Heading>

      <Text mb={4}>
        <Link
          href="https://github.com/garethgeorge/backrest"
          target="_blank"
          colorPalette="blue"
        >
          Check for new Backrest releases on GitHub
        </Link>
      </Text>

      <DividerWithText>Overview</DividerWithText>

      <List.Root gap={2} ml={5} as="ul" listStyleType="disc">
        <List.Item>
          Repos map directly to restic repositories, start by configuring your
          backup locations.
        </List.Item>
        <List.Item>
          Plans are where you configure directories to backup, and backup
          scheduling. Multiple plans can backup to a single restic repository.
        </List.Item>
        <List.Item>
          See{" "}
          <Link
            href="https://restic.readthedocs.io/en/latest/030_preparing_a_new_repo.html"
            target="_blank"
            colorPalette="blue"
          >
            the restic docs on preparing a new repository
          </Link>{" "}
          for details about available repository types and how they can be
          configured.
        </List.Item>
        <List.Item>
          See{" "}
          <Link
            href="https://garethgeorge.github.io/backrest"
            target="_blank"
            colorPalette="blue"
          >
            the Backrest wiki
          </Link>{" "}
          for instructions on how to configure Backrest.
        </List.Item>
      </List.Root>

      <DividerWithText>Tips</DividerWithText>

      <List.Root gap={2} ml={5} as="ul" listStyleType="disc">
        <List.Item>
          Backup your Backrest configuration: your Backrest config holds all of
          your repos, plans, and the passwords to decrypt them. When you have
          Backrest configured to your liking make sure to store a copy of your
          config (or minimally a copy of your passwords) in a safe location e.g.
          a secure note in your password manager.
        </List.Item>
        <List.Item>
          Configure hooks: Backrest can deliver notifications about backup
          events. It's strongly recommended that you configure an on error hook
          that will notify you in the event that backups start failing (e.g. an
          issue with storage or network connectivity). Hooks can be configured
          either at the plan or repo level.
        </List.Item>
      </List.Root>

      {isDevBuild && (
        <>
          <DividerWithText>Config View</DividerWithText>
          <AccordionRoot collapsible variant="plain">
            <AccordionItem value="config">
              <AccordionItemTrigger>
                Config JSON hidden for security
              </AccordionItemTrigger>
              <AccordionItemContent>
                {config ? (
                  <Box
                    as="pre"
                    p={2}
                    bg="gray.900"
                    color="white"
                    borderRadius="md"
                    fontSize="xs"
                    overflowX="auto"
                  >
                    {toJsonString(ConfigSchema, config, { prettySpaces: 2 })}
                  </Box>
                ) : (
                  <Spinner />
                )}
              </AccordionItemContent>
            </AccordionItem>
          </AccordionRoot>
        </>
      )}
    </Box>
  );
};
