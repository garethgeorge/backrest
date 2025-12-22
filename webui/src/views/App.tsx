import React, { Suspense, useEffect, useState } from "react";
import {
  ScheduleOutlined,
  DatabaseOutlined,
  PlusOutlined,
  CheckCircleOutlined,
  ExclamationOutlined,
  SettingOutlined,
  LoadingOutlined,
  CloudServerOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";
import { Button, Empty, Layout, Menu, Spin, theme } from "antd";
import { Config, Multihost_Peer } from "../../gen/ts/v1/config_pb";
import { useAlertApi } from "../components/Alerts";
import { useShowModal } from "../components/ModalManager";
import { uiBuildVersion } from "../state/buildcfg";
import { ActivityBar } from "../components/ActivityBar";
import { OperationEvent, OperationStatus } from "../../gen/ts/v1/operations_pb";
import {
  subscribeToOperations,
  unsubscribeFromOperations,
} from "../state/oplog";
import LogoSvg from "../../assets/logo.svg";
import { debounce, keyBy } from "../lib/util";
import { Code } from "@connectrpc/connect";
import { LoginModal } from "./LoginModal";
import { backrestService, setAuthToken } from "../api";
import { useConfig } from "../components/ConfigProvider";
import { shouldShowSettings } from "../state/configutil";
import { OpSelector, OpSelectorSchema } from "../../gen/ts/v1/service_pb";
import { colorForStatus } from "../state/flowdisplayaggregator";
import { getStatusForSelector, matchSelector } from "../state/logstate";
import { Route, Routes, useNavigate, useParams } from "react-router-dom";
import { MainContentAreaTemplate } from "./MainContentArea";
import { create } from "@bufbuild/protobuf";
import { PeerState, RepoMetadata } from "../../gen/ts/v1sync/syncservice_pb";
import { useSyncStates } from "../state/peerstates";
import * as m from "../paraglide/messages";
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

const SelectorView = React.lazy(() =>
  import("./SelectorView").then((m) => ({
    default: m.SelectorView,
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
      breadcrumbs={[{ title: m.app_breadcrumb_repo() }, { title: repoId! }]}
      key={repoId}
    >
      {repo ? (
        <RepoView repo={repo} />
      ) : (
        <Empty description={m.app_repo_not_found({ repoId: repoId || "" })} />
      )}
    </MainContentAreaTemplate>
  );
};

const RemoteRepoViewContainer = () => {
  const { peerInstanceId, repoId } = useParams();
  const peerStates = useSyncStates();

  // Peer state is used to find the right repo
  const peerState = peerStates.find(
    (state) => state.peerInstanceId === peerInstanceId
  );
  const peerRepo = (peerState?.knownRepos || []).find((r) => r.id === repoId);

  return (
    <MainContentAreaTemplate
      breadcrumbs={[
        { title: m.app_breadcrumb_peer() },
        { title: peerInstanceId || m.app_unknown_peer() },
        { title: m.app_breadcrumb_repo() },
        { title: repoId || m.app_unknown_repo() },
      ]}
      key={`${peerInstanceId}-${repoId}`}
    >
      {peerRepo ? (
        <SelectorView
          title={m.app_remote_repo_title({ id: peerRepo.id })}
          sel={create(OpSelectorSchema, {
            originalInstanceKeyid: peerState?.peerKeyid,
            repoGuid: peerRepo.guid,
          })}
        />
      ) : (
        <Empty description={m.app_repo_not_found({ repoId: repoId || "" })} />
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
      breadcrumbs={[{ title: m.app_breadcrumb_plan() }, { title: planId! }]}
      key={planId}
    >
      {plan ? (
        <PlanView plan={plan} />
      ) : (
        <Empty description={m.app_plan_not_found({ planId: planId || "" })} />
      )}
    </MainContentAreaTemplate>
  );
};

export const App: React.FC = () => {
  const {
    token: { colorBgContainer, colorTextLightSolid },
  } = theme.useToken();
  const navigate = useNavigate();
  const [config, setConfig] = useConfig();

  const peerStates = useSyncStates();

  const items = getSidenavItems(config, peerStates);

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
            {m.app_logout()}
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
        <AuthenticationBoundary>
          <Suspense fallback={<Spin />}>
            <Routes>
              <Route
                path="/"
                element={
                  <MainContentAreaTemplate breadcrumbs={[{ title: m.app_breadcrumb_summary() }]}>
                    <SummaryDashboard />
                  </MainContentAreaTemplate>
                }
              />
              <Route
                path="/getting-started"
                element={
                  <MainContentAreaTemplate
                    breadcrumbs={[{ title: m.app_breadcrumb_getting_started() }]}
                  >
                    <GettingStartedGuide />
                  </MainContentAreaTemplate>
                }
              />
              <Route path="/plan/:planId" element={<PlanViewContainer />} />
              <Route path="/repo/:repoId" element={<RepoViewContainer />} />
              <Route
                path="/peer/:peerInstanceId/repo/:repoId"
                element={<RemoteRepoViewContainer />}
              />
              <Route
                path="/*"
                element={
                  <MainContentAreaTemplate breadcrumbs={[]}>
                    <Empty description={m.app_page_not_found()} />
                  </MainContentAreaTemplate>
                }
              />
            </Routes>
          </Suspense>
        </AuthenticationBoundary>
      </Layout>
    </Layout>
  );
};

const AuthenticationBoundary = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const [config, setConfig] = useConfig();
  const alertApi = useAlertApi()!;
  const showModal = useShowModal();

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
        const code = err.code;
        if (err.code === Code.Unauthenticated) {
          showModal(<LoginModal />);
          return;
        } else if (
          err.code !== Code.Unavailable &&
          err.code !== Code.DeadlineExceeded
        ) {
          alertApi.error(err.message, 0);
          return;
        }

        alertApi.error(
          m.app_error_initial_config(),
          0
        );
      });
  }, []);

  if (!config) {
    return <></>;
  }

  return <>{children}</>;
};

const getSidenavItems = (
  config: Config | null,
  peerStates: PeerState[]
): MenuProps["items"] => {
  const showModal = useShowModal();
  const navigate = useNavigate();

  if (!config) {
    return;
  }

  const reposById = keyBy(config.repos, (r) => r.id);
  const configPlans = config.plans || [];
  const configRepos = config.repos || [];

  const menu: MenuProps["items"] = [];

  const plans: MenuProps["items"] = [
    {
      key: "add-plan",
      icon: <PlusOutlined />,
      label: m.app_menu_add_plan(),
      onClick: async () => {
        const { AddPlanModal } = await import("./AddPlanModal");
        showModal(<AddPlanModal template={null} />);
      },
    },
    ...configPlans.map((plan) => {
      const sel = create(OpSelectorSchema, {
        originalInstanceKeyid: "",
        planId: plan.id,
        repoGuid: reposById[plan.repo]?.guid,
      });

      return {
        key: "p-" + plan.id,
        icon: <IconForResource selector={sel} />,
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
      label: m.app_menu_add_repo(),
      onClick: async () => {
        const { AddRepoModal } = await import("./AddRepoModal");
        showModal(<AddRepoModal template={null} />);
      },
    },
    ...configRepos.map((repo) => {
      return {
        key: "r-" + repo.id,
        icon: (
          <IconForResource
            selector={create(OpSelectorSchema, {
              instanceId: config.instance,
              repoGuid: repo.guid,
            })}
          />
        ),
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

  const authorizedClients: MenuProps["items"] = [];
  if (config.multihost?.authorizedClients?.length) {
    const authorizedClientsConfigs = new Map<string, Multihost_Peer>();
    for (const client of config.multihost.authorizedClients) {
      authorizedClientsConfigs.set(client.keyid, client);
    }

    const createElementForPeerState = (
      peerState: PeerState,
      peerConfig: Multihost_Peer
    ): Required<MenuProps>["items"][0] => {
      const repos: MenuProps["items"] = peerState.knownRepos.map(
        (repo: RepoMetadata) => {
          const sel = create(OpSelectorSchema, {
            originalInstanceKeyid: peerState.peerKeyid,
            repoGuid: repo.guid,
          });

          return {
            key: `repo-${peerState.peerKeyid}-${repo.guid}`,
            icon: <IconForResource selector={sel} />,
            label: (
              <div
                className="backrest visible-on-hover"
                style={{ width: "100%", height: "100%" }}
              >
                {repo.id}
              </div>
            ),
            onClick: async () => {
              navigate(`/peer/${peerState.peerInstanceId}/repo/${repo.id}`);
            },
          };
        }
      );

      return {
        key: `peer-${peerState.peerKeyid}`,
        icon: (
          <IconForResource
            selector={create(OpSelectorSchema, {
              originalInstanceKeyid: peerState.peerKeyid,
            })}
          />
        ),
        label: (
          <div
            className="backrest visible-on-hover"
            style={{ width: "100%", height: "100%" }}
          >
            {peerState.peerInstanceId}
          </div>
        ),
        children: repos.length > 0 ? repos : undefined,
      };
    };

    for (const peerState of peerStates) {
      const peerConfig = authorizedClientsConfigs.get(peerState.peerKeyid);
      if (!peerConfig) {
        continue;
      }
      authorizedClients.push(createElementForPeerState(peerState, peerConfig));
    }
  }

  menu.push({
    key: "plans",
    icon: React.createElement(ScheduleOutlined),
    label: m.app_menu_plans(),
    children: plans,
  });
  menu.push({
    key: "repos",
    icon: React.createElement(DatabaseOutlined),
    label: m.app_menu_repos(),
    children: repos,
  });
  if (authorizedClients.length > 0) {
    menu.push({
      key: "authorized-clients",
      icon: React.createElement(CloudServerOutlined),
      label: m.app_menu_remote_instances(),
      children: authorizedClients,
    });
  }
  menu.push({
    key: "settings",
    icon: React.createElement(SettingOutlined),
    label: m.app_menu_settings(),
    onClick: async () => {
      const { SettingsModal } = await import("./SettingsModal");
      showModal(<SettingsModal />);
    },
  });
  return menu;
};

const IconForResource = ({ selector }: { selector: OpSelector }) => {
  const [status, setStatus] = useState(OperationStatus.STATUS_UNKNOWN);
  useEffect(() => {
    const load = async () => {
      setStatus(await getStatusForSelector(selector));
    };
    load();
    const refresh = debounce(load, 1000, { maxWait: 10000, trailing: true });
    const callback = (event?: OperationEvent, err?: Error) => {
      if (!event || !event.event) return;
      switch (event.event.case) {
        case "createdOperations":
        case "updatedOperations":
          const ops = event.event.value.operations;
          if (ops.find((op) => matchSelector(selector, op))) {
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
  }, [JSON.stringify(selector)]);
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
