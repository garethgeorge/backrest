import React, { useEffect, useState } from "react";
import {
  ScheduleOutlined,
  DatabaseOutlined,
  PlusOutlined,
  CheckCircleOutlined,
  PaperClipOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";
import { Button, Layout, List, Menu, Modal, Spin, theme } from "antd";
import { configState, fetchConfig } from "../state/config";
import { useRecoilState } from "recoil";
import { Config, Plan } from "../../gen/ts/v1/config.pb";
import { useAlertApi } from "../components/Alerts";
import { useShowModal } from "../components/ModalManager";
import { AddPlanModal } from "./AddPlanModel";
import { AddRepoModel } from "./AddRepoModel";
import { MainContentArea, useSetContent } from "./MainContentArea";
import { PlanView } from "./PlanView";
import {
  EOperation,
  buildOperationListListener,
  getOperations,
  subscribeToOperations,
  toEop,
  unsubscribeFromOperations,
} from "../state/oplog";
import { formatTime } from "../lib/formatting";
import { SnapshotBrowser } from "../components/SnapshotBrowser";
import { OperationRow } from "../components/OperationList";
import {
  Operation,
  OperationEvent,
  OperationEventType,
} from "../../gen/ts/v1/operations.pb";

const { Header, Content, Sider } = Layout;

export const App: React.FC = () => {
  const {
    token: { colorBgContainer, colorTextLightSolid },
  } = theme.useToken();

  const [config, setConfig] = useRecoilState(configState);
  const alertApi = useAlertApi()!;
  const showModal = useShowModal();
  const setContent = useSetContent();

  useEffect(() => {
    showModal(<Spin spinning={true} fullscreen />);

    fetchConfig()
      .then((config) => {
        setConfig(config);
        showModal(null);
      })
      .catch((err) => {
        alertApi.error(err.message, 0);
        alertApi.error(
          "Failed to fetch initial config, typically this means the UI could not connect to the backend"
        );
      });
  }, []);

  const items = getSidenavItems(config);

  return (
    <Layout style={{ height: "auto" }}>
      <Header style={{ display: "flex", alignItems: "center" }}>
        <h1>
          <a
            style={{ color: colorTextLightSolid }}
            onClick={() => setContent(null, [])}
          >
            ResticUI{" "}
          </a>
          <small style={{ color: "rgba(255,255,255,0.3)", fontSize: "0.3em" }}>
            {process.env.BUILD_TIME ? "v" + process.env.BUILD_TIME : ""}
          </small>
        </h1>
      </Header>
      <Layout>
        <Sider width={300} style={{ background: colorBgContainer }}>
          <Menu
            mode="inline"
            defaultSelectedKeys={["1"]}
            defaultOpenKeys={["plans", "repos"]}
            style={{ height: "100%", borderRight: 0 }}
            items={items}
          />
        </Sider>
        <MainContentArea />
      </Layout>
    </Layout>
  );
};

const getSidenavItems = (config: Config | null): MenuProps["items"] => {
  const showModal = useShowModal();
  const setContent = useSetContent();
  const [snapshotsByPlan, setSnapshotsByPlan] = useState<{
    [planId: string]: EOperation[];
  }>({});

  const addSnapshots = (planId: string, ops: EOperation[]) => {
    const snapsByPlanCpy = { ...snapshotsByPlan };
    let snapsForPlanCpy = [...(snapsByPlanCpy[planId] || [])];
    for (const op of ops) {
      snapsForPlanCpy.push(toEop(op));
    }
    snapsForPlanCpy.sort((a, b) => {
      return a.parsedTime > b.parsedTime ? -1 : 1;
    });
    if (snapsForPlanCpy.length > 5) {
      snapsForPlanCpy = snapsForPlanCpy.slice(0, 5);
    }
    snapsByPlanCpy[planId] = snapsForPlanCpy;
    setSnapshotsByPlan(snapsByPlanCpy);
  };

  // Track newly created snapshots in the set.
  useEffect(() => {
    const listener = (event: OperationEvent) => {
      if (event.type !== OperationEventType.EVENT_CREATED) return;
      const op = event.operation!;
      if (!op.planId) return;
      if (!op.operationIndexSnapshot) return;
      addSnapshots(op.planId!, [toEop(op)]);
    };

    subscribeToOperations(listener);

    return () => {
      unsubscribeFromOperations(listener);
    };
  }, [snapshotsByPlan]);

  if (!config) return [];

  const configPlans = config.plans || [];
  const configRepos = config.repos || [];

  const onSelectPlan = (plan: Plan) => {
    setContent(<PlanView plan={plan} />, [
      { title: "Plans" },
      { title: plan.id || "" },
    ]);

    if (!snapshotsByPlan[plan.id!]) {
      (async () => {
        const ops = await getOperations({ planId: plan.id!, lastN: "20" });
        // avoid races by checking again after the request
        if (!snapshotsByPlan[plan.id!]) {
          const snapshots = ops.filter((op) => !!op.operationIndexSnapshot);
          addSnapshots(plan.id!, snapshots);
        }
      })();
    }
  };

  const plans: MenuProps["items"] = [
    {
      key: "add-plan",
      icon: <PlusOutlined />,
      label: "Add Plan",
      onClick: () => {
        showModal(<AddPlanModal template={null} />);
      },
    },
    ...configPlans.map((plan) => {
      const children: MenuProps["items"] = (
        snapshotsByPlan[plan.id!] || []
      ).map((snapshot) => {
        return {
          key: "s-" + snapshot.id,
          icon: <PaperClipOutlined />,
          label: (
            <small>{"Operation " + formatTime(snapshot.parsedTime)}</small>
          ),
          onClick: () => {
            showModal(
              <Modal
                title="View Snapshot"
                open={true}
                onCancel={() => showModal(null)}
                footer={[
                  <Button
                    key="done"
                    onClick={() => showModal(null)}
                    type="primary"
                  >
                    Done
                  </Button>,
                ]}
              >
                <List>
                  <OperationRow operation={snapshot} />
                </List>
              </Modal>
            );
          },
        };
      });

      return {
        key: "p-" + plan.id,
        icon: <CheckCircleOutlined style={{ color: "green" }} />,
        label: plan.id,
        children: children,
        onTitleClick: onSelectPlan.bind(null, plan), // if children
        onClick: onSelectPlan.bind(null, plan), // if no children
      };
    }),
  ];

  const repos: MenuProps["items"] = [
    {
      key: "add-repo",
      icon: <PlusOutlined />,
      label: "Add Repo",
      onClick: () => {
        showModal(<AddRepoModel template={null} />);
      },
    },
    ...configRepos.map((repo) => {
      return {
        key: "r-" + repo.id,
        icon: <CheckCircleOutlined style={{ color: "green" }} />,
        label: repo.id,
        onClick: () => {
          showModal(<AddRepoModel template={repo} />);
        },
      };
    }),
  ];

  return [
    {
      key: "plans",
      icon: React.createElement(ScheduleOutlined),
      label: "Plans",
      children: plans,
    },
    {
      key: "repos",
      icon: React.createElement(DatabaseOutlined),
      label: "Repositories",
      children: repos,
    },
  ];
};
