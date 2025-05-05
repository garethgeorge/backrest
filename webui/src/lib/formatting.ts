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

const durationUnits = ["seconds", "minutes", "hours", "days"] as const;
type DurationUnit = typeof durationUnits[number];

export interface FormatDurationOptions {
  minUnit?: DurationUnit;
  maxUnit?: DurationUnit;
}


export const formatDuration = (ms: number, options?: FormatDurationOptions) => {
  const minUnitIndex = durationUnits.indexOf(options?.minUnit || "seconds");
  const maxUnitIndex = durationUnits.indexOf(options?.maxUnit || "hours");

  const seconds = Math.ceil(ms / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  let parts: string[] = [];

  if (maxUnitIndex >= 3 && days > 0) {
    parts.push(`${days}d`);
  }
  if (maxUnitIndex >= 2 && minUnitIndex <= 2) {
    const h = maxUnitIndex === 2 ? hours : (hours % 24);
    if (h > 0) {
      parts.push(`${h}h`);
    }
  }
  if (maxUnitIndex >= 1 && minUnitIndex <= 1) {
    const m = maxUnitIndex === 1 ? minutes : (minutes % 60);
    if (m > 0) {
      parts.push(`${m}m`);
    }
  }
  if (maxUnitIndex >= 0) {
    const s = maxUnitIndex === 0 ? seconds : (seconds % 60);
    if (s > 0 || parts.length === 0) {
      parts.push(`${s}s`);
    }
  }

  return parts.join("");
};

export const normalizeSnapshotId = (id: string) => {
  return id.substring(0, 8);
};
