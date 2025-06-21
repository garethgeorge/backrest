import React, { useState } from "react";
import { ConfigProvider, theme } from "antd";
import { useConfig } from "./ConfigProvider";

const darkTheme = window.matchMedia("(prefers-color-scheme: dark)").matches;

const customTheme = {
    token: {
        fontSize: 20,
        paddingXS: 16,
        lineHeight: 2
    }
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
    const themeConfig = {
        algorithm: algorithm,
        ...customTheme
    }
    return (
        <ConfigProvider
            theme={themeConfig}
        >
            {children}
        </ConfigProvider>
    )
};
