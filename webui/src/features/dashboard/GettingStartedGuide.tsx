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
import * as m from "../../paraglide/messages";

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
        {m.dashboard_getting_started_title()}
      </Heading>

      <Text mb={4}>
        <Link
          href="https://github.com/garethgeorge/backrest"
          target="_blank"
          colorPalette="blue"
        >
          {m.dashboard_getting_started_check()}
        </Link>
      </Text>

      <DividerWithText>
        {m.dashboard_getting_started_overview()}
      </DividerWithText>

      <List.Root gap={2} ml={5} as="ul" listStyleType="disc">
        <List.Item>{m.dashboard_getting_started_overview_a()}</List.Item>
        <List.Item>{m.dashboard_getting_started_overview_b()}</List.Item>
        <List.Item>
          {m.dashboard_getting_started_overview_c_a()}
          <Link
            href="https://restic.readthedocs.io/en/latest/030_preparing_a_new_repo.html"
            target="_blank"
            colorPalette="blue"
          >
            {m.dashboard_getting_started_overview_c_b()}
          </Link>
          {m.dashboard_getting_started_overview_c_c()}
        </List.Item>
        <List.Item>
          {m.dashboard_getting_started_overview_d_a()}
          <Link
            href="https://garethgeorge.github.io/backrest"
            target="_blank"
            colorPalette="blue"
          >
            {m.dashboard_getting_started_overview_d_b()}
          </Link>
          {m.dashboard_getting_started_overview_d_c()}
        </List.Item>
      </List.Root>

      <DividerWithText>{m.dashboard_getting_started_tips()}</DividerWithText>

      <List.Root gap={2} ml={5} as="ul" listStyleType="disc">
        <List.Item>{m.dashboard_getting_started_tips_a()}</List.Item>
        <List.Item>{m.dashboard_getting_started_tips_b()}</List.Item>
      </List.Root>

      {isDevBuild && (
        <>
          <DividerWithText>
            {m.dashboard_getting_started_config_view()}
          </DividerWithText>
          <AccordionRoot collapsible variant="plain">
            <AccordionItem value="config">
              <AccordionItemTrigger>
                {m.dashboard_getting_started_config_json()}
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
