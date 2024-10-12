import React from "react";
import { DisplayType, colorForStatus } from "../state/flowdisplayaggregator";
import {
  CodeOutlined,
  DeleteOutlined,
  DownloadOutlined,
  FileSearchOutlined,
  InfoCircleOutlined,
  PaperClipOutlined,
  RobotOutlined,
  SaveOutlined,
} from "@ant-design/icons";
import { OperationStatus } from "../../gen/ts/v1/operations_pb";

export const OperationIcon = ({
  type,
  status,
}: {
  type: DisplayType;
  status: OperationStatus;
}) => {
  const color = colorForStatus(status);

  let avatar: React.ReactNode;
  switch (type) {
    case DisplayType.BACKUP:
      avatar = <SaveOutlined style={{ color: color }} />;
      break;
    case DisplayType.FORGET:
      avatar = <DeleteOutlined style={{ color: color }} />;
      break;
    case DisplayType.SNAPSHOT:
      avatar = <PaperClipOutlined style={{ color: color }} />;
      break;
    case DisplayType.RESTORE:
      avatar = <DownloadOutlined style={{ color: color }} />;
      break;
    case DisplayType.PRUNE:
      avatar = <DeleteOutlined style={{ color: color }} />;
      break;
    case DisplayType.CHECK:
      avatar = <FileSearchOutlined style={{ color: color }} />;
    case DisplayType.RUNHOOK:
      avatar = <RobotOutlined style={{ color: color }} />;
      break;
    case DisplayType.STATS:
      avatar = <InfoCircleOutlined style={{ color: color }} />;
      break;
    case DisplayType.RUNCOMMAND:
      avatar = <CodeOutlined style={{ color: color }} />;
      break;
  }

  return avatar;
};
