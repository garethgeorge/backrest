import React, { useState } from "react";
import { PeerState, ConnectionState } from "../../gen/ts/v1sync/syncservice_pb";
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloseCircleOutlined,
  DisconnectOutlined,
  ExclamationCircleOutlined,
  KeyOutlined,
  LoadingOutlined,
  QuestionCircleOutlined,
} from "@ant-design/icons";
import { Tooltip } from "antd";

export const PeerStateConnectionStatusIcon = ({
  peerState,
}: {
  peerState: PeerState;
}) => {
  const getStatusIcon = () => {
    switch (peerState.state) {
      case ConnectionState.CONNECTED:
        return (
          <CheckCircleOutlined style={{ color: "#52c41a", fontSize: "16px" }} />
        );

      case ConnectionState.PENDING:
        return (
          <LoadingOutlined style={{ color: "#1890ff", fontSize: "16px" }} />
        );

      case ConnectionState.RETRY_WAIT:
        return (
          <ClockCircleOutlined style={{ color: "#faad14", fontSize: "16px" }} />
        );

      case ConnectionState.DISCONNECTED:
        return (
          <DisconnectOutlined style={{ color: "#d9d9d9", fontSize: "16px" }} />
        );

      case ConnectionState.ERROR_AUTH:
        return <KeyOutlined style={{ color: "#ff4d4f", fontSize: "16px" }} />;

      case ConnectionState.ERROR_PROTOCOL:
        return (
          <ExclamationCircleOutlined
            style={{ color: "#ff4d4f", fontSize: "16px" }}
          />
        );

      case ConnectionState.ERROR_INTERNAL:
        return (
          <CloseCircleOutlined style={{ color: "#ff4d4f", fontSize: "16px" }} />
        );

      case ConnectionState.UNKNOWN:
      default:
        return (
          <QuestionCircleOutlined
            style={{ color: "#8c8c8c", fontSize: "16px" }}
          />
        );
    }
  };

  const getStatusTooltip = () => {
    const baseMessage = `${peerState.peerInstanceId}: `;

    switch (peerState.state) {
      case ConnectionState.CONNECTED:
        return (
          baseMessage +
          "Connected" +
          (peerState.statusMessage ? ` - ${peerState.statusMessage}` : "")
        );
      case ConnectionState.PENDING:
        return (
          baseMessage +
          "Connecting..." +
          (peerState.statusMessage ? ` - ${peerState.statusMessage}` : "")
        );

      case ConnectionState.RETRY_WAIT:
        return (
          baseMessage +
          "Retrying connection" +
          (peerState.statusMessage ? ` - ${peerState.statusMessage}` : "")
        );

      case ConnectionState.DISCONNECTED:
        return (
          baseMessage +
          "Disconnected" +
          (peerState.statusMessage ? ` - ${peerState.statusMessage}` : "")
        );

      case ConnectionState.ERROR_AUTH:
        return (
          baseMessage +
          "Authentication error" +
          (peerState.statusMessage ? ` - ${peerState.statusMessage}` : "")
        );

      case ConnectionState.ERROR_PROTOCOL:
        return (
          baseMessage +
          "Protocol error" +
          (peerState.statusMessage ? ` - ${peerState.statusMessage}` : "")
        );

      case ConnectionState.ERROR_INTERNAL:
        return (
          baseMessage +
          "Internal error" +
          (peerState.statusMessage ? ` - ${peerState.statusMessage}` : "")
        );

      case ConnectionState.UNKNOWN:
      default:
        return (
          baseMessage +
          "Unknown status" +
          (peerState.statusMessage ? ` - ${peerState.statusMessage}` : "")
        );
    }
  };

  return (
    <Tooltip title={getStatusTooltip()} placement="top">
      <span
        style={{ cursor: "help", display: "inline-flex", alignItems: "center" }}
      >
        {getStatusIcon()}
      </span>
    </Tooltip>
  );
};
