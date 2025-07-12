import React, { Suspense, useContext, useEffect, useState } from "react";
import { Repo } from "../../gen/ts/v1/config_pb";
import { Flex, Tabs, Tooltip, Typography, Button } from "antd";
import { OperationListView } from "../components/OperationListView";
import { OperationTreeView } from "../components/OperationTreeView";
import { MAX_OPERATION_HISTORY, STATS_OPERATION_HISTORY } from "../constants";
import {
  GetOperationsRequestSchema,
  OpSelector,
} from "../../gen/ts/v1/service_pb";
import { useConfig } from "../components/ConfigProvider";
import { useShowModal } from "../components/ModalManager";
import { create } from "@bufbuild/protobuf";

export const SelectorView = ({
  title,
  sel,
}: React.PropsWithChildren<{ title: string; sel: OpSelector }>) => {
  console.log("SelectorView", title, sel);
  const items = [
    {
      key: "1",
      label: "Tree View",
      children: (
        <>
          <OperationTreeView
            req={create(GetOperationsRequestSchema, {
              selector: sel,
              lastN: BigInt(MAX_OPERATION_HISTORY),
            })}
          />
        </>
      ),
      destroyOnHidden: true,
    },
    {
      key: "2",
      label: "List View",
      children: (
        <>
          <h3>Backup Action History</h3>
          <OperationListView
            req={create(GetOperationsRequestSchema, {
              selector: sel,
              lastN: BigInt(MAX_OPERATION_HISTORY),
            })}
            showPlan={true}
            showDelete={true}
          />
        </>
      ),
      destroyOnHidden: true,
    },
  ];
  return (
    <>
      {title ? (
        <Flex gap="small" align="center" wrap="wrap">
          <Typography.Title>{title}</Typography.Title>
        </Flex>
      ) : null}
      <Tabs defaultActiveKey={items[0].key} items={items} />
    </>
  );
};
