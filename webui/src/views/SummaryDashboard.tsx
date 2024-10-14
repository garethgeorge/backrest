import { Collapse, Divider, Spin, Typography } from "antd";
import React, { useEffect, useState } from "react";
import { useConfig } from "../components/ConfigProvider";
import { useSetContent } from "./MainContentArea";
import { SummaryDashboardResponse } from "../../gen/ts/v1/service_pb";
import { backrestService } from "../api";
import { useAlertApi } from "../components/Alerts";

export const GettingStartedGuide = () => {
  const config = useConfig()[0];
  const setContent = useSetContent();
  const alertApi = useAlertApi()!;

  const [summaryData, setSummaryData] =
    useState<SummaryDashboardResponse | null>();

  useEffect(() => {
    // Fetch summary data
    const fetchData = async () => {
      // check if the tab is in the foreground
      if (document.hidden) {
        return;
      }

      try {
        const data = await backrestService.getSummaryDashboard({});
        setSummaryData(data);
      } catch (e) {
        alertApi.error("Failed to fetch summary data", e);
      }
    };

    fetchData();

    const interval = setInterval(fetchData, 60000);
    return () => clearInterval(interval);
  }, []);

  return (
    <>
      <Typography.Title level={2}>Dashboard</Typography.Title>
      <Divider />

      <
    </>
  );
};
