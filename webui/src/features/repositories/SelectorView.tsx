import React from "react";
import { Flex, Heading } from "@chakra-ui/react";
import {
  TabsRoot,
  TabsList,
  TabsTrigger,
  TabsContent,
} from "../../components/ui/tabs";
import { OperationListView } from "../operations/OperationListView";
import { OperationTreeView } from "../operations/OperationTreeView";
import { MAX_OPERATION_HISTORY } from "../../constants";
import {
  GetOperationsRequestSchema,
  OpSelector,
} from "../../../gen/ts/v1/service_pb";
import { create } from "@bufbuild/protobuf";

import * as m from "../../paraglide/messages";
export const SelectorView = ({
  title,
  sel,
}: React.PropsWithChildren<{ title: string; sel: OpSelector }>) => {
  return (
    <>
      {title ? (
        <Flex gap="small" align="center" wrap="wrap" mb={4}>
          <Heading size="xl">{title}</Heading>
        </Flex>
      ) : null}

      <TabsRoot defaultValue="tree" lazyMount unmountOnExit>
        <TabsList>
          <TabsTrigger value="tree" data-testid="view-tab-tree">
            {m.repo_tab_tree()}
          </TabsTrigger>
          <TabsTrigger value="list" data-testid="view-tab-list">
            {m.repo_tab_list()}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="tree">
          <OperationTreeView
            req={create(GetOperationsRequestSchema, {
              selector: sel,
              lastN: BigInt(MAX_OPERATION_HISTORY),
            })}
          />
        </TabsContent>

        <TabsContent value="list">
          <Heading size="md" mb={4}>
            {m.repo_history_title()}
          </Heading>
          <OperationListView
            req={create(GetOperationsRequestSchema, {
              selector: sel,
              lastN: BigInt(MAX_OPERATION_HISTORY),
            })}
            showPlan={true}
            showDelete={true}
          />
        </TabsContent>
      </TabsRoot>
    </>
  );
};
