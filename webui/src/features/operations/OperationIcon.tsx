import React from "react";
import { DisplayType, colorForStatus } from "../../api/flowDisplayAggregator";
import {
  FaCode,
  FaTrash,
  FaDownload,
  FaSearch,
  FaInfoCircle,
  FaPaperclip,
  FaRobot,
  FaSave,
} from "react-icons/fa";
import { OperationStatus } from "../../../gen/ts/v1/operations_pb";

export const OperationIcon = ({
  type,
  status,
}: {
  type: DisplayType;
  status: OperationStatus;
}) => {
  const color = colorForStatus(status);
  const style = { color: color };

  let avatar: React.ReactNode;
  switch (type) {
    case DisplayType.BACKUP:
    case DisplayType.BACKUP_DRYRUN:
      avatar = <FaSave style={style} />;
      break;
    case DisplayType.FORGET:
      avatar = <FaTrash style={style} />;
      break;
    case DisplayType.SNAPSHOT:
      avatar = <FaPaperclip style={style} />;
      break;
    case DisplayType.RESTORE:
      avatar = <FaDownload style={style} />;
      break;
    case DisplayType.PRUNE:
      avatar = <FaTrash style={style} />;
      break;
    case DisplayType.CHECK:
      avatar = <FaSearch style={style} />;
      break;
    case DisplayType.RUNHOOK:
      avatar = <FaRobot style={style} />;
      break;
    case DisplayType.STATS:
      avatar = <FaInfoCircle style={style} />;
      break;
    case DisplayType.RUNCOMMAND:
      avatar = <FaCode style={style} />;
      break;
  }

  return avatar;
};
