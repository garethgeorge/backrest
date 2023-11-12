import React, { useEffect } from "react";
import {
  ScheduleOutlined,
  DatabaseOutlined,
  PlusOutlined,
  CheckCircleOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";
import { Breadcrumb, Layout, Menu, Spin, message, theme } from "antd";
import { configState, fetchConfig } from "../state/config";
import { useRecoilState } from "recoil";
import { Config } from "../../gen/ts/v1/config.pb";
import { AlertContextProvider, useAlertApi } from "../components/Alerts";
import { useShowModal } from "../components/ModalManager";
import { AddPlanModal } from "./AddPlanModel";
import { AddRepoModel } from "./AddRepoModel";

const { Header, Content, Sider } = Layout;

export const App: React.FC = () => {
  const {
    token: { colorBgContainer, colorTextLightSolid },
  } = theme.useToken();

  const [config, setConfig] = useRecoilState(configState);
  const alertApi = useAlertApi()!;
  const showModal = useShowModal();

  useEffect(() => {
    showModal(<Spin spinning={true} fullscreen />);

    fetchConfig()
      .then((config) => {
        setConfig(config);
      })
      .catch((err) => {
        alertApi.error(err.message, 0);
      })
      .finally(() => {
        showModal(null);
      });
  }, []);

  console.log("rerender config ", config);
  const items = getSidenavItems(config);

  return (
    <Layout style={{ height: "100vh" }}>
      <Header style={{ display: "flex", alignItems: "center" }}>
        <h1 style={{ color: colorTextLightSolid }}>ResticUI</h1>
      </Header>
      <Layout>
        <Sider width={200} style={{ background: colorBgContainer }}>
          <Menu
            mode="inline"
            defaultSelectedKeys={["1"]}
            defaultOpenKeys={["plans", "repos"]}
            style={{ height: "100%", borderRight: 0 }}
            items={items}
          />
        </Sider>
        <Layout style={{ padding: "0 24px 24px" }}>
          <Breadcrumb
            style={{ margin: "16px 0" }}
            items={[{ title: "Home" }]}
          ></Breadcrumb>
          <Content
            style={{
              padding: 24,
              margin: 0,
              minHeight: 280,
              background: colorBgContainer,
            }}
          >
            Content
          </Content>
        </Layout>
      </Layout>
    </Layout>
  );
};

const getSidenavItems = (config: Config | null): MenuProps["items"] => {
  const showModal = useShowModal();

  if (!config) return [];

  const configPlans = config.plans || [];
  const configRepos = config.repos || [];

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
      return {
        key: "p-" + plan.id,
        icon: <CheckCircleOutlined style={{ color: "green" }} />,
        label: plan.id,
        onClick: () => {
          showModal(<AddPlanModal template={plan} />);
        },
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
