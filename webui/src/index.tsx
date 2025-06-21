import * as React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./views/App";
import { AlertContextProvider } from "./components/Alerts";
import { ModalContextProvider } from "./components/ModalManager";

import "react-js-cron/dist/styles.css";
import { ConfigContextProvider } from "./components/ConfigProvider";
import { HashRouter } from "react-router-dom";
import { CustomThemeProvider } from "./components/CustomThemeProvider";

const el = document.querySelector("#app");
el &&
  createRoot(el).render(
    <React.StrictMode>
      <ConfigContextProvider>
        <CustomThemeProvider>
          <AlertContextProvider>
            <ModalContextProvider>
              <HashRouter>
                <App />
              </HashRouter>
            </ModalContextProvider>
          </AlertContextProvider>
        </CustomThemeProvider>
      </ConfigContextProvider>
    </React.StrictMode>
  );
