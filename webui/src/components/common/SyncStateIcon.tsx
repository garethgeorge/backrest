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
import { Box } from "@chakra-ui/react";
import { Tooltip } from "../ui/tooltip";

import * as m from "../../paraglide/messages";
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
        statusText = m.sync_state_icon_connected();
        break;
      case ConnectionState.PENDING:
        statusText = m.sync_state_icon_connecting();
        break;
      case ConnectionState.RETRY_WAIT:
        statusText = m.sync_state_icon_retrying_connection();
        break;
      case ConnectionState.DISCONNECTED:
        statusText = m.sync_state_icon_disconnected();
        break;
      case ConnectionState.ERROR_AUTH:
        statusText = m.sync_state_icon_authentication_error();
        break;
      case ConnectionState.ERROR_PROTOCOL:
        statusText = m.sync_state_icon_protocol_error();
        break;
      case ConnectionState.ERROR_INTERNAL:
        statusText = m.sync_state_icon_internal_error();
        break;
      case ConnectionState.UNKNOWN:
      default:
        statusText = m.sync_state_icon_unknown_status();
        break;
    }

    return (
      baseMessage +
      statusText +
      (peerState.statusMessage ? ` - ${peerState.statusMessage}` : "")
    );
  };

  return (
    <Tooltip
      content={getStatusTooltip()}
      portalled
      showArrow
      positionerProps={{ zIndex: 2100 }}
    >
      <Box
        as="button"
        cursor="help"
        display="inline-flex"
        alignItems="center"
        bg="transparent"
        border="none"
        p={0}
        m={0}
        lineHeight={1}
      >
        {getStatusIcon()}
      </Box>
    </Tooltip>
  );
};
