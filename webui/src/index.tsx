import * as React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./views/App";

const el = document.querySelector("#app");
el && createRoot(el).render(<App />);
