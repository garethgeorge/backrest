const units = ["B", "KiB", "MiB", "GiB", "TiB", "PiB"];
export const formatBytes = (bytes?: number | string) => {
  if (!bytes) {
    return "0B";
  }
  if (typeof bytes === "string") {
    bytes = parseInt(bytes);
  }

  let unit = 0;
  while (bytes > 1024) {
    bytes /= 1024;
    unit++;
  }
  return `${Math.round(bytes * 100) / 100} ${units[unit]}`;
};

const fmtHourMinute = new Intl.DateTimeFormat(undefined, {
  hour: "2-digit",
  minute: "2-digit",
  day: "2-digit",
  month: "2-digit",
  year: "numeric",
});

const timezoneOffsetMs = new Date().getTimezoneOffset() * 60 * 1000;
// formatTime formats a time as YYYY-MM-DD at HH:MM AM/PM
export const formatTime = (time: number | string | Date) => {
  if (typeof time === "string") {
    time = parseInt(time);
  } else if (time instanceof Date) {
    time = time.getTime();
  }
  const d = new Date(time);
  return fmtHourMinute.format(d);
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
};

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
