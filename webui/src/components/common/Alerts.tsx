import React from "react";
import { toaster } from "../ui/toaster";

export const alerts = {
  error: (content: any, duration = 5000) => {
    const title =
      typeof content === "string" ? content : formatErrorAlert(content);
    toaster.create({
      title: title,
      type: "error",
      duration,
    });
  },
  success: (content: string, duration = 3000) => {
    toaster.create({
      title: content,
      type: "success",
      duration,
    });
  },
  info: (content: string, duration = 3000) => {
    toaster.create({
      title: content,
      type: "info",
      duration,
    });
  },
  warning: (content: string, duration = 4000) => {
    toaster.create({
      title: content,
      type: "warning",
      duration,
    });
  },
  loading: (content: string) => {
    return toaster.create({
      title: content,
      type: "loading",
    });
  },
  destroy: (key?: string) => {
    if (key) toaster.dismiss(key);
  },
};

export const formatErrorAlert = (error: any, prefix?: string) => {
  prefix = prefix ? prefix.trim() + " " : "Error: ";
  const contents = (error.message || "" + error) as string;
  if (contents.includes("\n")) {
    return (
      <>
        {prefix}
        <pre style={{ alignContent: "normal", textAlign: "left" }}>
          {contents}
        </pre>
      </>
    );
  }
  if (prefix.indexOf(":") === -1) {
    prefix += ":";
  }
  return `${prefix} ${contents}`;
};
