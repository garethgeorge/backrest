import * as React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./views/App";
import { AlertContextProvider } from "./components/Alerts";
import { ModalContextProvider } from "./components/ModalManager";

import "react-js-cron/dist/styles.css";
import { ConfigProvider as AntdConfigProvider, theme } from "antd";
import { ConfigContextProvider } from "./components/ConfigProvider";
import { MainContentProvider } from "./views/MainContentArea";
import { ThemeProvider, createTheme } from "@mui/material";

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

const darkThemeMq = window.matchMedia("(prefers-color-scheme: dark)");

const el = document.querySelector("#app");
el &&
  createRoot(el).render(
    <AntdConfigProvider
      theme={{
        algorithm: [
          darkThemeMq.matches ? theme.darkAlgorithm : theme.defaultAlgorithm,
          theme.compactAlgorithm,
        ],
      }}
    >
      <ThemeProvider theme={createTheme({
        palette: {
          mode: "dark",
        },
      })}>
        <Root>
          <App />
        </Root>
      </ThemeProvider>
    </AntdConfigProvider>
  );
