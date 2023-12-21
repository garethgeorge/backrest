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
// formatTime formats a time as YYYY-MM-DD at HH:MM AM/PM
export const formatTime = (time: number | string | Date) => {
  if (typeof time === "string") {
    time = parseInt(time);
  } else if (time instanceof Date) {
    time = time.getTime();
  }
  const d = new Date();
  d.setTime(time - timezoneOffsetMs);
  const isoStr = d.toISOString();
  const hours = d.getUTCHours() % 12 == 0 ? 12 : d.getUTCHours() % 12;
  const minutes =
    d.getUTCMinutes() < 10 ? "0" + d.getUTCMinutes() : d.getUTCMinutes();
  return `${isoStr.substring(0, 10)} at ${hours}:${minutes} ${
    d.getUTCHours() > 12 ? "PM" : "AM"
  }`;
};

export const localISOTime = (time: number | string | Date) => {
  if (typeof time === "string") {
    time = parseInt(time);
  } else if (time instanceof Date) {
    time = time.getTime();
  }

  const d = new Date();
  d.setTime(time - timezoneOffsetMs);
  return d.toISOString();
}

// formatDate formats a time as YYYY-MM-DD
export const formatDate = (time: number | string | Date) => {
  if (typeof time === "string") {
    time = parseInt(time);
  } else if (time instanceof Date) {
    time = time.getTime();
  }
  let d = new Date();
  d.setTime(time - timezoneOffsetMs);
  const isoStr = d.toISOString();
  return isoStr.substring(0, 10);
};

export const formatDuration = (ms: number) => {
  const seconds = Math.ceil(ms / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  if (hours === 0 && minutes === 0) {
    return `${seconds % 60}s`;
  } else if (hours === 0) {
    return `${minutes}m${seconds % 60}s`;
  }
  return `${hours}h${minutes % 60}m${Math.floor(seconds % 60)}s`;
};

export const normalizeSnapshotId = (id: string) => {
  return id.substring(0, 8);
};
