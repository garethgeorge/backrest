import React from "react";
import { Button } from "../../components/ui/button";
import {
  MenuRoot,
  MenuTrigger,
  MenuContent,
  MenuItem,
  MenuSeparator,
} from "../../components/ui/menu";
import { Flex, Box } from "@chakra-ui/react";
import { FiFilter, FiCheckSquare, FiSquare } from "react-icons/fi";
import {
  DisplayType,
  displayTypeToString,
} from "../../api/flowDisplayAggregator";
import { OperationIcon } from "./OperationIcon";
import { OperationStatus } from "../../../gen/ts/v1/operations_pb";

export interface OperationFilterProps {
  types: DisplayType[];
  activeTypes: Set<DisplayType>;
  onToggle: (type: DisplayType) => void;
  onSelectAll: () => void;
  onSelectNone: () => void;
}

export const OperationFilter = ({
  types,
  activeTypes,
  onToggle,
  onSelectAll,
  onSelectNone,
}: OperationFilterProps) => {
  return (
    <MenuRoot>
      <MenuTrigger asChild>
        <Button size="xs" variant="outline" colorPalette="blue">
          <Flex align="center" gap={1}>
            <FiFilter />
            <span style={{ fontSize: "0.75rem" }}>Filter</span>
            <span style={{ fontSize: "0.7rem", opacity: 0.8 }}>
              ({activeTypes.size}/{types.length})
            </span>
          </Flex>
        </Button>
      </MenuTrigger>
      <MenuContent minW="220px">
        <MenuItem value="all" closeOnSelect={false} onClick={onSelectAll}>
          <Flex align="center" gap={2} width="100%">
            <Box flex="1" fontSize="sm">
              All
            </Box>
            {activeTypes.size === types.length ? (
              <FiCheckSquare size={14} />
            ) : (
              <FiSquare size={14} />
            )}
          </Flex>
        </MenuItem>
        <MenuItem value="none" closeOnSelect={false} onClick={onSelectNone}>
          <Flex align="center" gap={2} width="100%">
            <Box flex="1" fontSize="sm">
              None
            </Box>
            {activeTypes.size === 0 ? (
              <FiCheckSquare size={14} />
            ) : (
              <FiSquare size={14} />
            )}
          </Flex>
        </MenuItem>
        <MenuSeparator />
        {types.map((type) => (
          <MenuItem
            key={type}
            value={String(type)}
            closeOnSelect={false}
            onClick={() => onToggle(type)}
          >
            <Flex align="center" gap={2} width="100%">
              <OperationIcon
                type={type}
                status={OperationStatus.STATUS_UNKNOWN}
              />
              <Box flex="1" fontSize="sm">
                {displayTypeToString(type)}
              </Box>
              {activeTypes.has(type) ? (
                <FiCheckSquare size={14} />
              ) : (
                <FiSquare size={14} />
              )}
            </Flex>
          </MenuItem>
        ))}
      </MenuContent>
    </MenuRoot>
  );
};
