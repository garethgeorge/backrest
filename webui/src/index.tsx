import * as React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./views/App";
import { AlertContextProvider } from "./components/Alerts";
import { ModalContextProvider } from "./components/ModalManager";

import "react-js-cron/dist/styles.css";
import { ConfigProvider as AntdConfigProvider, theme } from "antd";
import { ConfigContextProvider } from "./components/ConfigProvider";
import { MainContentProvider } from "./views/MainContentArea";
import { StyledEngineProvider, ThemeProvider, createTheme } from "@mui/material";

const Root = ({ children }: { children: React.ReactNode }) => {
  return (
    <ConfigContextProvider>
      <AlertContextProvider>
        <MainContentProvider>
          <ModalContextProvider>{children}</ModalContextProvider>
        </MainContentProvider>
      </AlertContextProvider>
    </ConfigContextProvider>
  );
};

const darkTheme = window.matchMedia("(prefers-color-scheme: dark)").matches;

const el = document.querySelector("#app");
el &&
  createRoot(el).render(
    <AntdConfigProvider
      theme={{
        algorithm: [
          darkTheme ? theme.darkAlgorithm : theme.defaultAlgorithm,
          theme.compactAlgorithm,
        ],
      }}
    >
      <StyledEngineProvider injectFirst>
        <ThemeProvider theme={createTheme({
          palette: {
            mode: darkTheme ? "dark" : "light"
          },
        })}>
          <Root>
            <App />
          </Root>
        </ThemeProvider>
      </StyledEngineProvider>
    </AntdConfigProvider>
  );
