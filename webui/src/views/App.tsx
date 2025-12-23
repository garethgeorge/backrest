import React, { Suspense, useEffect, useState } from "react";
import {
  FiCalendar,
  FiDatabase,
  FiPlus,
  FiCheckCircle,
  FiAlertTriangle,
  FiSettings,
  FiLoader,
  FiRadio,
  FiActivity, // Added as a placeholder/guess for ActivityBar if needed, or stick to component
  FiServer
} from "react-icons/fi";

import { Box, Flex, Button, Heading, Text, Spinner, Separator } from "@chakra-ui/react";
import { AccordionRoot, AccordionItem, AccordionItemTrigger, AccordionItemContent } from "../components/ui/accordion";
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
import { Route, Routes, useNavigate, useParams, Link as RouterLink } from "react-router-dom";
import { MainContentAreaTemplate } from "./MainContentArea";
import { create } from "@bufbuild/protobuf";
import { PeerState, RepoMetadata } from "../../gen/ts/v1sync/syncservice_pb";
import { useSyncStates } from "../state/peerstates";
import * as m from "../paraglide/messages";
import { Link } from "../components/ui/link";
import { EmptyState } from "../components/ui/empty-state";
import { ColorModeButton } from "../components/ui/color-mode";

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

// Wrappers for consistent views with breadcrumbs and error handling
const RepoViewContainer = () => {
  const { repoId } = useParams();
  const [config, setConfig] = useConfig();

  if (!config) {
    return <Box p={10}><Spinner /></Box>;
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
        <EmptyState title={m.app_repo_not_found({ repoId: repoId || "" })} />
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
        <EmptyState title={m.app_repo_not_found({ repoId: repoId || "" })} />
      )}
    </MainContentAreaTemplate>
  );
};

const PlanViewContainer = () => {
  const { planId } = useParams();
  const [config, setConfig] = useConfig();

  if (!config) {
    return <Box p={10}><Spinner /></Box>;
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
        <EmptyState title={m.app_plan_not_found({ planId: planId || "" })} />
      )}
    </MainContentAreaTemplate>
  );
};

const Sidebar = () => {
  const [config] = useConfig();
  const peerStates = useSyncStates();
  const showModal = useShowModal();
  const navigate = useNavigate();

  // Replicate getSidenavItems functionality with Chakra components
  if (!config) return null;

  const reposById = keyBy(config.repos, (r) => r.id);

  // Sort logic can be added here if needed, currently adhering to original order
  const configPlans = config.plans || [];
  const configRepos = config.repos || [];

  return (
    <Box w="300px" bg="bg.panel" borderRightWidth="1px" borderColor="border" h="full" overflowY="auto" flexShrink={0}>
      <AccordionRoot multiple defaultValue={["plans", "repos", "authorized-clients"]} variant="plain">
        {/* PLANS SECTION */}
        <AccordionItem value="plans">
          <AccordionItemTrigger px={4} py={2} _hover={{ bg: "bg.subtle" }}>
            <Flex align="center" gap={2}>
              <FiCalendar />
              <Text fontWeight="medium">{m.app_menu_plans()}</Text>
            </Flex>
          </AccordionItemTrigger>
          <AccordionItemContent pb={2}>
            <Button
              variant="ghost"
              size="sm"
              width="full"
              justifyContent="flex-start"
              onClick={async () => {
                const { AddPlanModal } = await import("./AddPlanModal");
                showModal(<AddPlanModal template={null} />);
              }}
              pl={9}
              mb={1}
            >
              <FiPlus /> {m.app_menu_add_plan()}
            </Button>
            {configPlans.map(plan => {
              const sel = create(OpSelectorSchema, {
                originalInstanceKeyid: "",
                planId: plan.id,
                repoGuid: reposById[plan.repo]?.guid,
              });
              return (
                <Flex key={plan.id} align="center" pl={9} pr={2} py={1} _hover={{ bg: "bg.subtle" }} role="group" borderRadius="md" mx={2}>
                  <Box flexShrink={0} mr={2}>
                    <IconForResource selector={sel} />
                  </Box>
                  <Box flex="1" cursor="pointer" onClick={() => navigate(`/plan/${plan.id}`)} userSelect="none">
                    <Text truncate>{plan.id}</Text>
                  </Box>
                  <Box opacity={0} _groupHover={{ opacity: 1 }} transition="opacity 0.2s">
                    <Button
                      size="xs"
                      variant="ghost"
                      onClick={async (e) => {
                        e.stopPropagation();
                        const { AddPlanModal } = await import("./AddPlanModal");
                        showModal(<AddPlanModal template={plan} />);
                      }}
                    >
                      <FiSettings />
                    </Button>
                  </Box>
                </Flex>
              );
            })}
          </AccordionItemContent>
        </AccordionItem>

        {/* REPOS SECTION */}
        <AccordionItem value="repos">
          <AccordionItemTrigger px={4} py={2} _hover={{ bg: "bg.subtle" }}>
            <Flex align="center" gap={2}>
              <FiDatabase />
              <Text fontWeight="medium">{m.app_menu_repos()}</Text>
            </Flex>
          </AccordionItemTrigger>
          <AccordionItemContent pb={2}>
            <Button
              variant="ghost"
              size="sm"
              width="full"
              justifyContent="flex-start"
              onClick={async () => {
                const { AddRepoModal } = await import("./AddRepoModal");
                showModal(<AddRepoModal template={null} />);
              }}
              pl={9}
              mb={1}
            >
              <FiPlus /> {m.app_menu_add_repo()}
            </Button>
            {configRepos.map(repo => {
              return (
                <Flex key={repo.id} align="center" pl={9} pr={2} py={1} _hover={{ bg: "bg.subtle" }} role="group" borderRadius="md" mx={2}>
                  <Box flexShrink={0} mr={2}>
                    <IconForResource selector={create(OpSelectorSchema, {
                      instanceId: config.instance,
                      repoGuid: repo.guid,
                    })}
                    />
                  </Box>
                  <Box flex="1" cursor="pointer" onClick={() => navigate(`/repo/${repo.id}`)} userSelect="none">
                    <Text truncate>{repo.id}</Text>
                  </Box>
                  <Box opacity={0} _groupHover={{ opacity: 1 }} transition="opacity 0.2s">
                    <Button
                      size="xs"
                      variant="ghost"
                      onClick={async (e) => {
                        e.stopPropagation();
                        const { AddRepoModal } = await import("./AddRepoModal");
                        showModal(<AddRepoModal template={repo} />);
                      }}
                    >
                      <FiSettings />
                    </Button>
                  </Box>
                </Flex>
              );
            })}
          </AccordionItemContent>
        </AccordionItem>

        {/* REMOTE INSTANCES / AUTHORIZED CLIENTS */}
        {config.multihost?.authorizedClients?.length ? (
          <AccordionItem value="authorized-clients">
            <AccordionItemTrigger px={4} py={2} _hover={{ bg: "bg.subtle" }}>
              <Flex align="center" gap={2}>
                <FiServer />
                <Text fontWeight="medium">{m.app_menu_remote_instances()}</Text>
              </Flex>
            </AccordionItemTrigger>
            <AccordionItemContent pb={2}>
              {peerStates.map(peerState => {
                // Logic to get peer config if needed, filtering handled by original logic
                // Assuming we list all peerStates derived from hook
                const sel = create(OpSelectorSchema, { originalInstanceKeyid: peerState.peerKeyid });

                return (
                  <Box key={peerState.peerKeyid} mb={2}>
                    <Flex align="center" pl={9} pr={2} py={1}>
                      <Box flexShrink={0} mr={2}><IconForResource selector={sel} /></Box>
                      <Text fontWeight="bold" fontSize="sm">{peerState.peerInstanceId}</Text>
                    </Flex>

                    {/* Nested Repos for Peer */}
                    {peerState.knownRepos.map((repo: RepoMetadata) => (
                      <Flex
                        key={repo.guid}
                        align="center"
                        pl={12}
                        pr={2}
                        py={1}
                        _hover={{ bg: "bg.subtle" }}
                        borderRadius="md"
                        mx={2}
                        cursor="pointer"
                        onClick={() => navigate(`/peer/${peerState.peerInstanceId}/repo/${repo.id}`)}
                      >
                        <Box flexShrink={0} mr={2}>
                          <IconForResource selector={create(OpSelectorSchema, {
                            originalInstanceKeyid: peerState.peerKeyid,
                            repoGuid: repo.guid,
                          })} />
                        </Box>
                        <Text fontSize="sm" truncate>{repo.id}</Text>
                      </Flex>
                    ))}
                  </Box>
                )
              })}
            </AccordionItemContent>
          </AccordionItem>
        ) : null}

        {/* SETTINGS */}
        <Box mt={4} mx={4}>
          <Button
            variant="outline"
            size="sm"
            width="full"
            justifyContent="flex-start"
            onClick={async () => {
              const { SettingsModal } = await import("./SettingsModal");
              showModal(<SettingsModal />);
            }}
          >
            <FiSettings /> {m.app_menu_settings()}
          </Button>
        </Box>

      </AccordionRoot>
    </Box>
  )
}


export const App: React.FC = () => {
  const navigate = useNavigate();
  const [config, setConfig] = useConfig();

  return (
    <Flex direction="column" minH="100vh">
      {/* HEADER */}
      <Flex
        as="header"
        align="center"
        px={4}
        h="60px"
        bg="#1b232c" // Maintain original brand color
        color="white"
        flexShrink={0}
      >
        <Box
          as="a"
          cursor="pointer"
          onClick={() => navigate("/")}
          mr={4}
        >
          <img
            src={LogoSvg}
            style={{ height: "30px", marginBottom: "-4px" }}
          />
        </Box>

        <Flex align="baseline" gap={4}>
          <Link href="https://github.com/garethgeorge/backrest" target="_blank" color="whiteAlpha.700" fontSize="xs">
            {uiBuildVersion}
          </Link>
          <Box fontSize="xs">
            <ActivityBar />
          </Box>
        </Flex>

        <Flex ml="auto" align="center" gap={4}>
          <Text fontSize="xs" color="whiteAlpha.600">
            {config && config.instance ? config.instance : undefined}
          </Text>
          <ColorModeButton color="white" />
          {config && !config.auth?.disabled && (
            <Button
              variant="ghost"
              size="sm"
              color="white"
              _hover={{ bg: "whiteAlpha.200" }}
              onClick={() => {
                setAuthToken("");
                window.location.reload();
              }}
            >
              {m.app_logout()}
            </Button>
          )}
        </Flex>
      </Flex>

      {/* MAIN LAYOUT */}
      <Flex flex="1" overflow="hidden">
        {/* SIDEBAR */}
        <Sidebar />

        {/* CONTENT AREA */}
        <Box flex="1" overflowY="auto" bg="bg.canvas">
          <AuthenticationBoundary>
            <Suspense fallback={<Box p={10}><Spinner /></Box>}>
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
                      <EmptyState title="404" description={m.app_page_not_found()} />
                    </MainContentAreaTemplate>
                  }
                />
              </Routes>
            </Suspense>
          </AuthenticationBoundary>
        </Box>
      </Flex>
    </Flex>
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
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const timeoutPromise = new Promise((_, reject) =>
      setTimeout(() => reject(new Error("Request timed out, backend may be unavailable")), 5000)
    );

    Promise.race([
      backrestService.getConfig({}),
      timeoutPromise,
    ])
      // @ts-ignore
      .then((config: Config) => {
        setConfig(config);
        if (shouldShowSettings(config)) {
          import("./SettingsModal").then(({ SettingsModal }) => {
            showModal(<SettingsModal />);
          });
        } else {
          showModal(null);
        }
        setIsLoading(false);
      })
      .catch((err) => {
        setIsLoading(false);
        const code = err.code;
        if (err.code === Code.Unauthenticated) {
          showModal(<LoginModal />);
          return;
        } else if (
          err.code !== Code.Unavailable &&
          err.code !== Code.DeadlineExceeded
        ) {
          setError(err.message);
          alertApi.error(err.message, 0);
          return;
        }

        setError(m.app_error_initial_config());
        alertApi.error(
          m.app_error_initial_config(),
          0
        );
      });
  }, []);

  if (isLoading) {
    return (
      <Box p={10} display="flex" justifyContent="center">
        <Spinner size="xl" />
      </Box>
    );
  }

  if (error && !config) {
    return (
      <EmptyState
        title="Failed to load configuration"
        description={error}
        icon={<FiAlertTriangle />}
      >
        <Button onClick={() => window.location.reload()}>Retry</Button>
      </EmptyState>
    );
  }

  if (!config) {
    return <></>;
  }

  return <>{children}</>;
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
      return <FiAlertTriangle style={{ color }} />;
    case OperationStatus.STATUS_WARNING:
      return <FiAlertTriangle style={{ color }} />; // Using AlertTriangle for warning too
    case OperationStatus.STATUS_INPROGRESS:
      return <FiLoader style={{ color }} />;
    case OperationStatus.STATUS_UNKNOWN:
      return <FiLoader style={{ color }} />;
    default:
      return <FiCheckCircle style={{ color }} />;
  }
};
