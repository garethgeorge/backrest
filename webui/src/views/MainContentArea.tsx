import { Breadcrumb, Layout, Spin, theme } from "antd";
import { Content } from "antd/es/layout/layout";
import React, { useState } from "react";

interface Breadcrumb {
  title: string;
  onClick?: () => void;
}

interface ContentAreaState {
  content: React.ReactNode | null;
  breadcrumbs: Breadcrumb[];
}

type ContentAreaCtx = [
  ContentAreaState,
  (content: React.ReactNode, breadcrumbs: Breadcrumb[]) => void
];

const ContentAreaContext = React.createContext<ContentAreaCtx>([
  {
    content: null,
    breadcrumbs: [],
  },
  (content, breadcrumbs) => {},
]);

export const MainContentProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const [state, setState] = useState<ContentAreaState>({
    content: null,
    breadcrumbs: [],
  });

  return (
    <>
      <ContentAreaContext.Provider
        value={[
          state,
          (content, breadcrumbs) => {
            setState({ content, breadcrumbs });
          },
        ]}
      >
        {children}
      </ContentAreaContext.Provider>
    </>
  );
};

export const useSetContent = () => {
  const context = React.useContext(ContentAreaContext);
  return context[1];
};

export const MainContentArea = () => {
  const { breadcrumbs, content } = React.useContext(ContentAreaContext)[0];

  return (
    <MainContentAreaTemplate breadcrumbs={breadcrumbs}>
      {content ? (
        content
      ) : (
        <Spin size="large" style={{ display: "block", margin: "auto" }} />
      )}
    </MainContentAreaTemplate>
  );
};

export const MainContentAreaTemplate = ({
  breadcrumbs,
  children,
}: {
  breadcrumbs: Breadcrumb[];
  children: React.ReactNode;
}) => {
  const {
    token: { colorBgContainer },
  } = theme.useToken();

  return (
    <Layout style={{ padding: "0 24px 24px" }}>
      <Breadcrumb
        style={{ margin: "16px 0" }}
        items={[...(breadcrumbs || [])]}
      ></Breadcrumb>
      <Content
        style={{
          padding: 24,
          margin: 0,
          minHeight: 280,
          background: colorBgContainer,
        }}
      >
        {children}
      </Content>
    </Layout>
  );
};
