import * as React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./views/App";
import { RecoilRoot } from "recoil";
import { AlertContextProvider } from "./components/Alerts";
import { ModalContextProvider } from "./components/ModalManager";

import "react-js-cron/dist/styles.css";
import { ConfigProvider, theme } from "antd";

const Root = ({ children }: { children: React.ReactNode }) => {
  return (
    <RecoilRoot>
      <AlertContextProvider>
        <ModalContextProvider>{children}</ModalContextProvider>
      </AlertContextProvider>
    </RecoilRoot>
  );
};

const darkThemeMq = window.matchMedia("(prefers-color-scheme: dark)");

const el = document.querySelector("#app");
el &&
  createRoot(el).render(
    <ConfigProvider
      theme={{
        algorithm: [
          darkThemeMq ? theme.darkAlgorithm : theme.defaultAlgorithm,
          theme.compactAlgorithm,
        ],
      }}
    >
      <Root>
        <App />
      </Root>
    </ConfigProvider>
  );
