import React, { useState } from "react";
import { ConfigProvider, theme } from "antd";
import { useConfig } from "./ConfigProvider";
import { Config } from "../../gen/ts/v1/config_pb";

const darkTheme = window.matchMedia("(prefers-color-scheme: dark)").matches;

const customTheme = {
    token: {
        fontSize: 20,
        paddingXS: 16,
        lineHeight: 2
    }
}

const parseCustomTokens = (config: Config | null) => {
    if (config?.ui?.tokens) {
        try {
            return JSON.parse(config.ui.tokens)
        } catch (e) {
            return null
        }
    }
    return null
}

export const CustomThemeProvider = ({
    children,
}: {
    children: React.ReactNode;
}) => {
    const [config] = useConfig()
    let algorithm = [
        darkTheme ? theme.darkAlgorithm : theme.defaultAlgorithm
    ]
    if (config?.ui?.useCompactUi) {
        algorithm.push(theme.compactAlgorithm)
    }
    const customTokens = parseCustomTokens(config)
    const themeConfig = customTokens ? {
        algorithm: algorithm,
        ...customTokens
    } : {
        algorithm: algorithm
    }
    return (
        <ConfigProvider
            theme={themeConfig}
        >
            {children}
        </ConfigProvider>
    )
};
