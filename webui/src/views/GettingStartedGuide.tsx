import { Collapse, Divider, Typography } from "antd";
import React from "react";
import { useRecoilValue } from "recoil";
import { configState } from "../state/config";

export const GettingStartedGuide = () => {
  const config = useRecoilValue(configState);

  return (
    <>
      <Typography.Text>
        <h1>Getting Started</h1>
        <Divider orientation="left">Overview</Divider>
        <ul>
          <li>
            Repos map directly to restic repositories, start by configuring your
            backup locations.
          </li>
          <li>
            Plans are where you configure directories to backup, and backup
            scheduling. Multiple plans can backup to a single restic repository.
          </li>
          <li>
            See{" "}
            <a href="https://restic.readthedocs.io/en/latest/030_preparing_a_new_repo.html">
              the restic docs on preparing a new repository
            </a>{" "}
            for details about available repository types and how they can be
            configured.
          </li>
        </ul>
        <Divider orientation="left">Tips</Divider>
        <ul>
          <li>
            Backup your Backrest configuration: your Backrest config holds all of
            your repos, plans, and the passwords to decrypt them. When you have
            Backrest configured to your liking make sure to store a copy of your
            config (or minimally a copy of your passwords) in a safe location
            e.g. a secure note in your password manager.
          </li>
        </ul>
        <Divider orientation="left">Config View</Divider>
        <Collapse
          size="small"
          items={[
            {
              key: "1",
              label: "Config JSON hidden for security",
              children: (
                <Typography>
                  <pre>{config.toJsonString({
                    prettySpaces: 2,
                  })}</pre>
                </Typography>
              ),
            },
          ]}
        />
      </Typography.Text>
    </>
  );
};
