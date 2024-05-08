import { Button, Input, Modal, Space } from "antd";
import React from "react";
import { useShowModal } from "../components/ModalManager";
import { backrestService } from "../api";
import { SpinButton } from "../components/SpinButton";
import { ConnectError } from "@connectrpc/connect";


interface Invocation {
  command: string;
  output: string;
  error: string;
}


export const RunCommandModal = ({ repoId }: { repoId: string }) => {
  const showModal = useShowModal();
  const [command, setCommand] = React.useState("");
  const [running, setRunning] = React.useState(false);
  const [invocations, setInvocations] = React.useState<Invocation[]>([]);

  const handleCancel = () => {
    showModal(null);
  }

  const doExecute = async () => {
    if (running) return;
    setRunning(true);

    const newInvocation = { command, output: "", error: "" };
    setInvocations((invocations) => [newInvocation, ...invocations]);

    let segments: string[] = [];

    try {
      for await (const bytes of backrestService.runCommand({ repoId, command })) {
        const output = new TextDecoder("utf-8").decode(bytes.value);
        segments.push(output);
        setInvocations((invocations) => {
          const copy = [...invocations];
          copy[0] = {
            ...copy[0],
            output: segments.join(""),
          }
          return copy;
        });
      }
    } catch (e: any) {
      setInvocations((invocations) => {
        const copy = [...invocations];
        copy[0] = {
          ...copy[0],
          error: (e as Error).message,
        }
        return copy;
      });
    } finally {
      setRunning(false);
    }
  }

  return <Modal
    open={true}
    onCancel={handleCancel}
    title={"Run Command in repo " + repoId}
    width="80vw"
    footer={[]}
  >
    <Space.Compact style={{ width: '100%' }}>
      <Input placeholder="Run a restic comamnd e.g. 'help' to print help text" onChange={(e) => setCommand(e.target.value)} onKeyUp={(e) => {
        if (e.key === "Enter") {
          doExecute();
        }
      }} />
      <SpinButton type="primary" onClickAsync={doExecute}>Execute</SpinButton>
    </Space.Compact>
    {invocations.map((invocation, i) => <div key={i}>
      {invocation.output ? <pre>{invocation.output}</pre> : null}
      {invocation.error ? <pre style={{ color: "red" }}>{invocation.error}</pre> : null}
    </div>)}
  </Modal>
}