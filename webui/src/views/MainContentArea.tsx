import { Breadcrumb, Layout, Spin, theme } from "antd";
import { Content } from "antd/es/layout/layout";
import React from "react";

interface BreadcrumbItem {
  title: string;
  onClick?: () => void;
}

export const MainContentAreaTemplate = ({
  breadcrumbs,
  children,
}: {
  breadcrumbs: BreadcrumbItem[];
  children: React.ReactNode;
}) => {
  const {
    token: { colorBgContainer },
  } = theme.useToken();

  return (
    <Layout style={{ padding: "0 24px 24px" }}>
      <Breadcrumb
        style={{ margin: "16px 0" }}
        items={breadcrumbs.map((b) => ({ title: b.title, onClick: b.onClick }))}
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
