export const uios = (process.env.UI_OS || "").trim().toLowerCase();
export const isWindows = uios === "windows";
export const uiBuildVersion = (
  process.env.BACKREST_BUILD_VERSION || "dev-snapshot-build"
).trim();
export const isDevBuild = uiBuildVersion === "dev-snapshot-build";
