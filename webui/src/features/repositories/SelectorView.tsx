import React from "react";
import { Flex, Heading } from "@chakra-ui/react";
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from "../../components/ui/tabs";
import { OperationListView } from "../operations/OperationListView";
import { OperationTreeView } from "../operations/OperationTreeView";
import { MAX_OPERATION_HISTORY } from "../../constants";
import {
  GetOperationsRequestSchema,
  OpSelector,
} from "../../../gen/ts/v1/service_pb";
import { create } from "@bufbuild/protobuf";

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
          <TabsTrigger value="tree">Tree View</TabsTrigger>
          <TabsTrigger value="list">List View</TabsTrigger>
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
          <Heading size="md" mb={4}>Backup Action History</Heading>
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
