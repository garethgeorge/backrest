import React, { useContext } from "react";

import { message } from "antd";
import { MessageInstance } from "antd/es/message/interface";

const MessageContext = React.createContext<MessageInstance | null>(null);

export const AlertContextProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const [messageApi, contextHolder] = message.useMessage();

  return (
    <>
      {contextHolder}
      <MessageContext.Provider value={messageApi}>
        {children}
      </MessageContext.Provider>
    </>
  );
};

export const useAlertApi = () => {
  return useContext(MessageContext);
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
  return `${prefix}: ${contents}`;
};
