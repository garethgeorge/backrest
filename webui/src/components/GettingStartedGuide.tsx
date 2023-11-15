import React from "react";

export const GettingStartedGuide = () => {
  return (
    <>
      <h1>Getting Started</h1>
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
    </>
  );
};
