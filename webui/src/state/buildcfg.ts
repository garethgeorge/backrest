export const uios = (process.env.UI_OS || "").trim().toLowerCase();
export const isWindows = uios === "windows";
export const uiBuildVersion = (
  process.env.RESTICUI_BUILD_VERSION || "dev"
).trim();
