import * as React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./views/App";
import { AlertContextProvider } from "./components/Alerts";
import { ModalContextProvider } from "./components/ModalManager";

import "react-js-cron/dist/styles.css";
import { ConfigProvider as AntdConfigProvider, theme } from "antd";
import { ConfigContextProvider } from "./components/ConfigProvider";
import { HashRouter } from "react-router-dom";

const Root = ({ children }: { children: React.ReactNode }) => {
  return (
    <ConfigContextProvider>
      <AlertContextProvider>
        <ModalContextProvider>{children}</ModalContextProvider>
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
      <React.StrictMode>
        <Root>
          <HashRouter>
            <App />
          </HashRouter>
        </Root>
      </React.StrictMode>
    </AntdConfigProvider>
  );
