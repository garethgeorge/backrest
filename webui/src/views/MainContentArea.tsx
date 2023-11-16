import { Breadcrumb, Layout, Spin, theme } from "antd";
import { Content } from "antd/es/layout/layout";
import React from "react";
import { atom, useRecoilValue, useSetRecoilState } from "recoil";
import { GettingStartedGuide } from "./GettingStartedGuide";

interface Breadcrumb {
  title: string;
  onClick?: () => void;
}

const contentPanel = atom<React.ReactNode | null>({
  key: "ui.content",
  default: null,
});

const breadcrumbs = atom<Breadcrumb[]>({
  key: "ui.breadcrumbs",
  default: [],
});

export const useSetContent = () => {
  const setContent = useSetRecoilState(contentPanel);
  const setBreadcrumbs = useSetRecoilState(breadcrumbs);

  return (content: React.ReactNode | null, breadcrumbs: Breadcrumb[]) => {
    setContent(content);
    setBreadcrumbs(breadcrumbs);
  };
};

export const MainContentArea = () => {
  const {
    token: { colorBgContainer },
  } = theme.useToken();

  let content = useRecoilValue(contentPanel);
  let crumbs = useRecoilValue(breadcrumbs);

  if (!content) {
    content = <GettingStartedGuide />;
    crumbs = [{ title: "Getting started" }];
  }

  return (
    <Layout style={{ padding: "0 24px 24px" }}>
      <Breadcrumb style={{ margin: "16px 0" }} items={[...crumbs]}></Breadcrumb>
      <Content
        style={{
          padding: 24,
          margin: 0,
          minHeight: 280,
          background: colorBgContainer,
        }}
      >
        {content}
      </Content>
    </Layout>
  );
};
