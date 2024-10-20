import React, { Suspense, useEffect, useState } from "react";
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
import { Button, Empty, Layout, Menu, Spin, theme } from "antd";
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
import { useConfig } from "../components/ConfigProvider";
import { shouldShowSettings } from "../state/configutil";
import { OpSelector } from "../../gen/ts/v1/service_pb";
import { colorForStatus } from "../state/flowdisplayaggregator";
import { getStatusForSelector } from "../state/logstate";
import {
  createHashRouter,
  Route,
  RouterProvider,
  Routes,
  useNavigate,
  useParams,
} from "react-router-dom";
import { MainContentAreaTemplate } from "./MainContentArea";

const { Header, Sider } = Layout;

const SummaryDashboard = React.lazy(() =>
  import("./SummaryDashboard").then((m) => ({
    default: m.SummaryDashboard,
  }))
);

const GettingStartedGuide = React.lazy(() =>
  import("./GettingStartedGuide").then((m) => ({
    default: m.GettingStartedGuide,
  }))
);

const PlanView = React.lazy(() =>
  import("./PlanView").then((m) => ({
    default: m.PlanView,
  }))
);

const RepoView = React.lazy(() =>
  import("./RepoView").then((m) => ({
    default: m.RepoView,
  }))
);

const RepoViewContainer = () => {
  const { repoId } = useParams();
  const [config, setConfig] = useConfig();

  if (!config) {
    return <Spin />;
  }

  const repo = config.repos.find((r) => r.id === repoId);

  return (
    <MainContentAreaTemplate
      breadcrumbs={[{ title: "Repo" }, { title: repoId! }]}
      key={repoId}
    >
      {repo ? (
        <RepoView repo={repo} />
      ) : (
        <Empty description={`Repo ${repoId} not found`} />
      )}
    </MainContentAreaTemplate>
  );
};

const PlanViewContainer = () => {
  const { planId } = useParams();
  const [config, setConfig] = useConfig();

  if (!config) {
    return <Spin />;
  }

  const plan = config.plans.find((p) => p.id === planId);
  return (
    <MainContentAreaTemplate
      breadcrumbs={[{ title: "Plan" }, { title: planId! }]}
      key={planId}
    >
      {plan ? (
        <PlanView plan={plan} />
      ) : (
        <Empty description={`Plan ${planId} not found`} />
      )}
    </MainContentAreaTemplate>
  );
};

export const App: React.FC = () => {
  const {
    token: { colorBgContainer, colorTextLightSolid },
  } = theme.useToken();
  const alertApi = useAlertApi()!;
  const showModal = useShowModal();
  const navigate = useNavigate();
  const [config, setConfig] = useConfig();

  useEffect(() => {
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

  const items = getSidenavItems(config);

  if (!config) {
    return <Spin />;
  }

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
        <a
          style={{ color: colorTextLightSolid }}
          onClick={() => {
            navigate("/");
          }}
        >
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
        <Routes>
          <Route
            path="/"
            element={
              <MainContentAreaTemplate breadcrumbs={[{ title: "Summary" }]}>
                <Suspense fallback={<Spin />}>
                  <SummaryDashboard />
                </Suspense>
              </MainContentAreaTemplate>
            }
          />
          <Route
            path="/getting-started"
            element={
              <Suspense fallback={<Spin />}>
                <GettingStartedGuide />
              </Suspense>
            }
          />
          <Route
            path="/plan/:planId"
            element={
              <Suspense fallback={<Spin />}>
                <PlanViewContainer />
              </Suspense>
            }
          />
          <Route
            path="/repo/:repoId"
            element={
              <Suspense fallback={<Spin />}>
                <RepoViewContainer />
              </Suspense>
            }
          />
        </Routes>
      </Layout>
    </Layout>
  );
};

const getSidenavItems = (config: Config | null): MenuProps["items"] => {
  const showModal = useShowModal();
  const navigate = useNavigate();

  if (!config) {
    return;
  }

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
          <div
            className="backrest visible-on-hover"
            style={{ width: "100%", height: "100%" }}
          >
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
          navigate(`/plan/${plan.id}`);
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
          <div
            className="backrest visible-on-hover"
            style={{ width: "100%", height: "100%" }}
          >
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
          navigate(`/repo/${repo.id}`);
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
