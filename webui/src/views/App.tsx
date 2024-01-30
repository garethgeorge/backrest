import React, { useEffect, useState } from "react";
import {
  ScheduleOutlined,
  DatabaseOutlined,
  PlusOutlined,
  CheckCircleOutlined,
  ExclamationOutlined,
  SettingOutlined,
  LoadingOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";
import { Button, Layout, Menu, Spin, theme } from "antd";
import { configState, fetchConfig } from "../state/config";
import { useRecoilState } from "recoil";
import { Config } from "../../gen/ts/v1/config_pb";
import { useAlertApi } from "../components/Alerts";
import { useShowModal } from "../components/ModalManager";
import { MainContentArea, useSetContent } from "./MainContentArea";
import { uiBuildVersion } from "../state/buildcfg";
import { ActivityBar } from "../components/ActivityBar";
import { OperationStatus } from "../../gen/ts/v1/operations_pb";
import { colorForStatus, getStatusForPlan, getStatusForRepo, subscribeToOperations, unsubscribeFromOperations } from "../state/oplog";
import LogoSvg from "url:../../assets/logo.svg";
import _ from "lodash";
import { Code } from "@connectrpc/connect";
import { LoginModal } from "./LoginModal";

const { Header, Sider } = Layout;

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
        if (err.code) {
          const code = err.code;
          if (code === Code.Unauthenticated) {
            showModal(<LoginModal />)
            return;
          } else if (code === Code.Unavailable || code === Code.DeadlineExceeded) {
            alertApi.error(
              "Failed to fetch initial config, typically this means the UI could not connect to the backend",
              0
            );
            return;
          }
        }

        alertApi.error(err.message, 0);
        alertApi.error(
          "Failed to fetch initial config, typically this means the UI could not connect to the backend",
          0
        );
      });
  }, []);

  const items = getSidenavItems(config);

  return (
    <Layout style={{ height: "auto", minHeight: "100vh" }}>
      <Header
        style={{
          display: "flex",
          alignItems: "center",
          width: "100vw",
          height: "60px",
          backgroundColor: "#1b232c",
        }}
      >
        <a
          style={{ color: colorTextLightSolid }}
          onClick={() => setContent(null, [])}
        >
          <img src={LogoSvg} style={{ height: "30px", color: "white", marginBottom: "-8px", paddingRight: "10px", }} />
        </a>
        <h1>
          <small style={{ color: "rgba(255,255,255,0.3)", fontSize: "0.6em" }}>
            {uiBuildVersion}
          </small>
          <small style={{ fontSize: "0.6em", marginLeft: "30px" }}>
            <ActivityBar />
          </small>
        </h1>
        <h1 style={{ position: "absolute", right: "20px" }}>
          <small style={{ color: "rgba(255,255,255,0.3)", fontSize: "0.6em" }}>
            {config.host ? "Host: " + config.host : undefined}
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
        icon: <IconForResource planId={plan.id} />,
        label: (
          <div className="backrest visible-on-hover">
            {plan.id}{" "}
            <Button
              className="hidden-child"
              type="text"
              size="small"
              shape="circle"
              icon={<SettingOutlined />}
              onClick={async () => {
                const { AddPlanModal } = await import("./AddPlanModal");
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
        icon: <IconForResource repoId={repo.id} />,
        label: (
          <div className="backrest visible-on-hover">
            {repo.id}{" "}
            <Button
              className="hidden-child"
              type="text"
              size="small"
              shape="circle"
              icon={<SettingOutlined />}
              onClick={async () => {
                const { AddRepoModal } = await import("./AddRepoModal");
                showModal(<AddRepoModal template={repo} />);
              }}
            />
          </div>
        ),
        onClick: async () => {
          const { RepoView } = await import("./RepoView");

          setContent(<RepoView repo={repo} />, [
            { title: "Repos" },
            { title: repo.id || "" },
          ]);
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
    {
      key: "settings",
      icon: React.createElement(SettingOutlined),
      label: "Settings",
      onClick: async () => {
        const { SettingsModal } = await import("./SettingsModal");
        showModal(<SettingsModal />);
      }
    },
  ];
};

const IconForResource = ({ planId, repoId }: { planId?: string, repoId?: string }) => {
  const [status, setStatus] = useState(OperationStatus.STATUS_UNKNOWN);
  useEffect(() => {
    const load = async () => {
      if (planId) {
        setStatus(await getStatusForPlan(planId));
      } else if (repoId) {
        setStatus(await getStatusForRepo(repoId));
      }
    };
    load();
    const refresh = _.debounce(load, 5000);
    subscribeToOperations(refresh);
    return () => {
      unsubscribeFromOperations(refresh);
    }
  }, [planId, repoId]);
  return iconForStatus(status);
}

const iconForStatus = (status: OperationStatus) => {
  const color = colorForStatus(status);
  switch (status) {
    case OperationStatus.STATUS_ERROR:
      return <ExclamationOutlined style={{ color }} />;
    case OperationStatus.STATUS_WARNING:
      return <ExclamationOutlined style={{ color }} />;
    case OperationStatus.STATUS_INPROGRESS:
      return <LoadingOutlined style={{ color }} />
    case OperationStatus.STATUS_UNKNOWN:
      return <LoadingOutlined style={{ color }} />;
    default:
      return <CheckCircleOutlined style={{ color }} />;
  }
}
