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
import { Config } from "../../gen/ts/v1/config_pb";
import { useAlertApi } from "../components/Alerts";
import { useShowModal } from "../components/ModalManager";
import { uiBuildVersion } from "../state/buildcfg";
import { ActivityBar } from "../components/ActivityBar";
import { OperationEvent, OperationStatus } from "../../gen/ts/v1/operations_pb";
import {
  subscribeToOperations,
  unsubscribeFromOperations,
} from "../state/oplog";
import LogoSvg from "url:../../assets/logo.svg";
import _ from "lodash";
import { Code } from "@connectrpc/connect";
import { LoginModal } from "./LoginModal";
import { backrestService, setAuthToken } from "../api";
import { MainContentArea, useSetContent } from "./MainContentArea";
import { GettingStartedGuide } from "./GettingStartedGuide";
import { useConfig } from "../components/ConfigProvider";
import { shouldShowSettings } from "../state/configutil";
import { OpSelector } from "../../gen/ts/v1/service_pb";
import { colorForStatus } from "../state/flowdisplayaggregator";
import { getStatusForSelector } from "../state/logstate";

const { Header, Sider } = Layout;

export const App: React.FC = () => {
  const {
    token: { colorBgContainer, colorTextLightSolid },
  } = theme.useToken();
  const alertApi = useAlertApi()!;
  const showModal = useShowModal();
  const setContent = useSetContent();
  const [config, setConfig] = useConfig();

  useEffect(() => {
    showModal(<Spin spinning={true} fullscreen />);

    backrestService
      .getConfig({})
      .then((config) => {
        setConfig(config);
        if (shouldShowSettings(config)) {
          import("./SettingsModal").then(({ SettingsModal }) => {
            showModal(<SettingsModal />);
          });
        } else {
          showModal(null);
        }
      })
      .catch((err) => {
        if (err.code) {
          const code = err.code;
          if (code === Code.Unauthenticated) {
            showModal(<LoginModal />);
            return;
          } else if (
            code === Code.Unavailable ||
            code === Code.DeadlineExceeded
          ) {
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

  const showGettingStarted = () => {
    setContent(
      <React.Suspense fallback={<Spin />}>
        <GettingStartedGuide />
      </React.Suspense>,
      [
        {
          title: "Getting Started",
        },
      ]
    );
  };

  const showSummaryDashboard = async () => {
    const { SummaryDashboard } = await import("./SummaryDashboard");
    setContent(
      <React.Suspense fallback={<Spin />}>
        <SummaryDashboard />
      </React.Suspense>,
      [
        {
          title: "Summary Dashboard",
        },
      ]
    );
  };

  useEffect(() => {
    if (config === null) {
      setContent(<p>Loading...</p>, []);
    } else {
      showSummaryDashboard();
    }
  }, [config === null]);

  const items = getSidenavItems(config);

  return (
    <Layout style={{ height: "auto", minHeight: "100vh" }}>
      <Header
        style={{
          display: "flex",
          alignItems: "center",
          width: "100%",
          height: "60px",
          backgroundColor: "#1b232c",
        }}
      >
        <a style={{ color: colorTextLightSolid }} onClick={showGettingStarted}>
          <img
            src={LogoSvg}
            style={{
              height: "30px",
              color: "white",
              marginBottom: "-8px",
              paddingRight: "10px",
            }}
          />
        </a>
        <h1>
          <a href="https://github.com/garethgeorge/backrest" target="_blank">
            <small
              style={{ color: "rgba(255,255,255,0.3)", fontSize: "0.6em" }}
            >
              {uiBuildVersion}
            </small>
          </a>
          <small style={{ fontSize: "0.6em", marginLeft: "30px" }}>
            <ActivityBar />
          </small>
        </h1>
        <h1 style={{ position: "absolute", right: "20px" }}>
          <small style={{ color: "rgba(255,255,255,0.3)", fontSize: "0.6em" }}>
            {config && config.instance ? config.instance : undefined}
          </small>
          <Button
            type="text"
            style={{
              marginLeft: "10px",
              color: "white",
              visibility: config?.auth?.disabled ? "hidden" : "visible",
            }}
            onClick={() => {
              setAuthToken("");
              window.location.reload();
            }}
          >
            Logout
          </Button>
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
        icon: <IconForResource planId={plan.id} repoId={plan.repo} />,
        label: (
          <div className="backrest visible-on-hover">
            {plan.id}{" "}
            <Button
              className="hidden-child float-center-right"
              type="text"
              size="small"
              shape="circle"
              style={{ width: "30px", height: "30px" }}
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

          setContent(<PlanView key={plan.id} plan={plan} />, [
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
              type="text"
              size="small"
              shape="circle"
              className="hidden-child float-center-right"
              style={{ width: "30px", height: "30px" }}
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

          setContent(<RepoView key={repo.id} repo={repo} />, [
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
      },
    },
  ];
};

const IconForResource = ({
  planId,
  repoId,
}: {
  planId?: string;
  repoId?: string;
}) => {
  const [status, setStatus] = useState(OperationStatus.STATUS_UNKNOWN);
  useEffect(() => {
    const load = async () => {
      setStatus(await getStatusForSelector(new OpSelector({ planId, repoId })));
    };
    load();
    const refresh = _.debounce(load, 1000, { maxWait: 10000, trailing: true });
    const callback = (event?: OperationEvent, err?: Error) => {
      if (!event || !event.event) return;
      switch (event.event.case) {
        case "createdOperations":
        case "updatedOperations":
          const ops = event.event.value.operations;
          if (
            ops.find(
              (op) => (!planId || op.planId === planId) && op.repoId === repoId
            )
          ) {
            refresh();
          }
          break;
        case "deletedOperations":
          refresh();
          break;
      }
    };

    subscribeToOperations(callback);
    return () => {
      unsubscribeFromOperations(callback);
    };
  }, [planId, repoId]);
  return iconForStatus(status);
};

const iconForStatus = (status: OperationStatus) => {
  const color = colorForStatus(status);
  switch (status) {
    case OperationStatus.STATUS_ERROR:
      return <ExclamationOutlined style={{ color }} />;
    case OperationStatus.STATUS_WARNING:
      return <ExclamationOutlined style={{ color }} />;
    case OperationStatus.STATUS_INPROGRESS:
      return <LoadingOutlined style={{ color }} />;
    case OperationStatus.STATUS_UNKNOWN:
      return <LoadingOutlined style={{ color }} />;
    default:
      return <CheckCircleOutlined style={{ color }} />;
  }
};
