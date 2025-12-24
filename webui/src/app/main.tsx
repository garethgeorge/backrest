import * as React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { ModalContextProvider } from "../components/common/ModalManager";
import { Toaster } from "../components/ui/toaster";

import "react-js-cron/dist/styles.css";

import { HashRouter } from "react-router-dom";
import { AppProvider } from "./provider";

const Root = ({ children }: { children: React.ReactNode }) => {
  return (
    <AppProvider>
      <Toaster />
      <ModalContextProvider>{children}</ModalContextProvider>
    </AppProvider>
  );
};

const darkTheme = window.matchMedia("(prefers-color-scheme: dark)").matches;

const el = document.querySelector("#app");
el &&
  createRoot(el).render(
    <React.StrictMode>
      <Root>
        <HashRouter>
          <App />
        </HashRouter>
      </Root>
    </React.StrictMode>,
  );
