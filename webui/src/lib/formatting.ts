export const formatBytes = (bytes?: number | string) => {
  if (!bytes) {
    return 0;
  }
  if (typeof bytes === "string") {
    bytes = parseInt(bytes);
  }

  const units = ["B", "KB", "MB", "GB", "TB", "PB"];
  let unit = 0;
  while (bytes > 1024) {
    bytes /= 1024;
    unit++;
  }
  return `${Math.round(bytes * 100) / 100} ${units[unit]}`;
};

const timezoneOffsetMs = new Date().getTimezoneOffset() * 60 * 1000;
export const formatTime = (time: number | string) => {
  if (typeof time === "string") {
    time = parseInt(time);
  }
  const d = new Date();
  d.setTime(time - timezoneOffsetMs);
  const isoStr = d.toISOString();
  return `${isoStr.substring(0, 10)} ${d.getUTCHours()}h${d.getUTCMinutes()}m`;
};

export const formatDuration = (ms: number) => {
  const seconds = Math.floor(ms / 100) / 10;
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  if (hours === 0 && minutes === 0) {
    return `${seconds % 60}s`;
  } else if (hours === 0) {
    return `${minutes}m${seconds % 60}s`;
  }
  return `${hours}h${minutes % 60}m${seconds % 60}s`;
};

export const normalizeSnapshotId = (id: string) => {
  return id.substring(0, 8);
};
