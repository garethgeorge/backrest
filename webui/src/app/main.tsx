import * as React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { AlertContextProvider } from "../components/common/Alerts";
import { ModalContextProvider } from "../components/common/ModalManager";

import "react-js-cron/dist/styles.css";
import { ConfigProvider as AntdConfigProvider, theme } from "antd";
import { HashRouter } from "react-router-dom";
import { AppProvider } from "./provider";


const Root = ({ children }: { children: React.ReactNode }) => {
  return (
    <AppProvider>
      <AlertContextProvider>
        <ModalContextProvider>{children}</ModalContextProvider>
      </AlertContextProvider>
    </AppProvider>
  );
};

const darkTheme = window.matchMedia("(prefers-color-scheme: dark)").matches;

const el = document.querySelector("#app");
el &&
  createRoot(el).render(
    <React.StrictMode>
      <AntdConfigProvider
        theme={{
          algorithm: [
            darkTheme ? theme.darkAlgorithm : theme.defaultAlgorithm,
            theme.compactAlgorithm,
          ],
        }}
      >
        <Root>
          <HashRouter>
            <App />
          </HashRouter>
        </Root>
      </AntdConfigProvider>
    </React.StrictMode>
  );
