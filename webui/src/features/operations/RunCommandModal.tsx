import React from "react";
import { useShowModal } from "../../components/common/ModalManager";
import { backrestService } from "../../api/client";
import { alerts } from "../../components/common/Alerts";
import {
  GetOperationsRequestSchema,
  RunCommandRequestSchema,
} from "../../../gen/ts/v1/service_pb";
import { RepoProps } from "../../state/peerStates";
import { OperationListView } from "./OperationListView";
import { create } from "@bufbuild/protobuf";
import { useConfig } from "../../app/provider";
import { FormModal } from "../../components/common/FormModal";
import { Button } from "../../components/ui/button";
import { Input, Group, Stack, Text, Box } from "@chakra-ui/react";
import * as m from "../../paraglide/messages";

export const RunCommandModal = ({ repo }: { repo: RepoProps }) => {
  const [config] = useConfig();
  const showModal = useShowModal();
  const [command, setCommand] = React.useState("");
  const [running, setRunning] = React.useState(false);

  const handleCancel = () => {
    showModal(null);
  };

  const doExecute = async () => {
    if (!command) return;
    setRunning(true);

    const toRun = command.trim();
    setCommand("");

    try {
      await backrestService.runCommand(
        create(RunCommandRequestSchema, {
          repoId: repo.id!,
          command: toRun,
        }),
      );
    } catch (e: any) {
      alerts.error("Command failed: " + e.message);
    } finally {
      setRunning(false);
    }
  };

  return (
    <FormModal
      isOpen={true}
      onClose={handleCancel}
      title={`Run Command in repo ${repo.id}`}
      size="large"
      footer={
        <Button variant="ghost" onClick={handleCancel}>
          {m.button_close()}
        </Button>
      }
    >
      <Stack gap={4}>
        <Group attached>
          <Input
            placeholder="Run a restic command e.g. 'help' to print help text"
            value={command}
            onChange={(e) => setCommand(e.target.value)}
            onKeyUp={(e) => {
              if (e.key === "Enter") {
                doExecute();
              }
            }}
          />
          <Button loading={running} onClick={doExecute}>
            Execute
          </Button>
        </Group>

        {running && command && (
          <Text color="gray.500" fontStyle="italic">
            Warning: another command is already running. Wait for it to finish
            before running another operation that requires the repo lock.
          </Text>
        )}

        <Box>
          <OperationListView
            req={create(GetOperationsRequestSchema, {
              selector: {
                instanceId: config?.instance,
                repoGuid: repo.guid,
                planId: "_system_", // run commands are not associated with a plan
              },
            })}
            filter={(op) => op.op.case === "operationRunCommand"}
          />
        </Box>
      </Stack>
    </FormModal>
  );
};
