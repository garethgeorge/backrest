import React, { useState } from "react";
import {
  PeerState,
  ConnectionState,
} from "../../../gen/ts/v1sync/syncservice_pb";
import {
  FiCheckCircle,
  FiClock,
  FiXCircle,
  FiWifiOff,
  FiAlertCircle,
  FiKey,
  FiLoader,
  FiHelpCircle,
} from "react-icons/fi";
import { Box, Spinner, Text } from "@chakra-ui/react";
import { Tooltip } from "../ui/tooltip";

export const PeerStateConnectionStatusIcon = ({
  peerState,
}: {
  peerState: PeerState;
}) => {
  const getStatusIcon = () => {
    switch (peerState.state) {
      case ConnectionState.CONNECTED:
        return (
          <FiCheckCircle
            style={{
              color: "var(--chakra-colors-green-500)",
              fontSize: "16px",
            }}
          />
        );

      case ConnectionState.PENDING:
        return (
          <Box animation="spin 1s linear infinite">
            <FiLoader
              style={{
                color: "var(--chakra-colors-blue-500)",
                fontSize: "16px",
              }}
            />
          </Box>
        );

      case ConnectionState.RETRY_WAIT:
        return (
          <FiClock
            style={{
              color: "var(--chakra-colors-orange-400)",
              fontSize: "16px",
            }}
          />
        );

      case ConnectionState.DISCONNECTED:
        return (
          <FiWifiOff
            style={{ color: "var(--chakra-colors-gray-400)", fontSize: "16px" }}
          />
        );

      case ConnectionState.ERROR_AUTH:
        return (
          <FiKey
            style={{ color: "var(--chakra-colors-red-500)", fontSize: "16px" }}
          />
        );

      case ConnectionState.ERROR_PROTOCOL:
        return (
          <FiAlertCircle
            style={{ color: "var(--chakra-colors-red-500)", fontSize: "16px" }}
          />
        );

      case ConnectionState.ERROR_INTERNAL:
        return (
          <FiXCircle
            style={{ color: "var(--chakra-colors-red-500)", fontSize: "16px" }}
          />
        );

      case ConnectionState.UNKNOWN:
      default:
        return (
          <FiHelpCircle
            style={{ color: "var(--chakra-colors-gray-500)", fontSize: "16px" }}
          />
        );
    }
  };

  const getStatusTooltip = () => {
    const baseMessage = `${peerState.peerInstanceId}: `;
    let statusText = "";

    switch (peerState.state) {
      case ConnectionState.CONNECTED:
        statusText = "Connected";
        break;
      case ConnectionState.PENDING:
        statusText = "Connecting...";
        break;
      case ConnectionState.RETRY_WAIT:
        statusText = "Retrying connection";
        break;
      case ConnectionState.DISCONNECTED:
        statusText = "Disconnected";
        break;
      case ConnectionState.ERROR_AUTH:
        statusText = "Authentication error";
        break;
      case ConnectionState.ERROR_PROTOCOL:
        statusText = "Protocol error";
        break;
      case ConnectionState.ERROR_INTERNAL:
        statusText = "Internal error";
        break;
      case ConnectionState.UNKNOWN:
      default:
        statusText = "Unknown status";
        break;
    }

    return (
      baseMessage +
      statusText +
      (peerState.statusMessage ? ` - ${peerState.statusMessage}` : "")
    );
  };

  return (
    <Tooltip content={getStatusTooltip()}>
      <Box as="span" cursor="help" display="inline-flex" alignItems="center">
        {getStatusIcon()}
      </Box>
    </Tooltip>
  );
};
