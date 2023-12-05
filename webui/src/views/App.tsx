import React, { useEffect, useState } from "react";
import {
  ScheduleOutlined,
  DatabaseOutlined,
  PlusOutlined,
  CheckCircleOutlined,
  SettingOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";
import { Button, Layout, Menu, Spin, theme } from "antd";
import { configState, fetchConfig } from "../state/config";
import { useRecoilState, useRecoilValue } from "recoil";
import { Config } from "../../gen/ts/v1/config.pb";
import { useAlertApi } from "../components/Alerts";
import { useShowModal } from "../components/ModalManager";
import { MainContentArea, useSetContent } from "./MainContentArea";
import { AddPlanModal } from "./AddPlanModal";

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
          "Failed to fetch initial config, typically this means the UI could not connect to the backend",
          0
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
            BackRest<span style={{ color: "grey" }}>ic</span>{" "}
          </a>
          <small style={{ color: "rgba(255,255,255,0.3)", fontSize: "0.6em" }}>
            {process.env.RESTICUI_BUILD_VERSION
              ? process.env.RESTICUI_BUILD_VERSION
              : ""}
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

  if (!config) return [];

  const configPlans = config.plans || [];
  const configRepos = config.repos || [];

  const plans: MenuProps["items"] = [
    {
      key: "add-plan",
      icon: <PlusOutlined />,
      label: "Add Plan",
      onClick: async () => {
        const { AddPlanModal } = await import("./AddPlanModal");
        showModal(<AddPlanModal template={null} />);
      },
    },
    ...configPlans.map((plan) => {
      return {
        key: "p-" + plan.id,
        icon: <CheckCircleOutlined style={{ color: "green" }} />,
        label: (
          <div className="resticui visible-on-hover">
            {plan.id}{" "}
            <Button
              className="hidden-child"
              type="text"
              size="small"
              shape="circle"
              icon={<SettingOutlined />}
              onClick={() => {
                showModal(<AddPlanModal template={plan} />);
              }}
            />
          </div>
        ),
        onClick: async () => {
          const { PlanView } = await import("./PlanView");

          setContent(<PlanView plan={plan} />, [
            { title: "Plans" },
            { title: plan.id || "" },
          ]);
        },
      };
    }),
  ];

  const repos: MenuProps["items"] = [
    {
      key: "add-repo",
      icon: <PlusOutlined />,
      label: "Add Repo",
      onClick: async () => {
        const { AddRepoModal } = await import("./AddRepoModal");

        showModal(<AddRepoModal template={null} />);
      },
    },
    ...configRepos.map((repo) => {
      return {
        key: "r-" + repo.id,
        icon: <CheckCircleOutlined style={{ color: "green" }} />,
        label: repo.id,
        onClick: async () => {
          const { AddRepoModal } = await import("./AddRepoModal");

          showModal(<AddRepoModal template={repo} />);
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
