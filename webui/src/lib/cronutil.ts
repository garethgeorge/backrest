import { converter } from 'react-js-cron';

interface FindOptimalStartIndexConstraints {
  /** The number of forward values to consider when optimizing the start index */
  windowSize?: number;
  /** Whether the values at the end of the list can wrap around to the beginning */
  wrap?: boolean;
}

/**
 * Find the index of the value that is nearest to other values within the specified constraints.
 * @param sortedValues A list of numbers in ascending order
 * @param maxValue The maximum value that could appear in the list, e.g. 59 for a list of minutes
 * @returns The index of the value that is closest to the optimal start index
 */
const findMinimalDurationStartIndex = (
  sortedValues: readonly number[],
  maxValue: number,
  { windowSize = 0, wrap = false }: FindOptimalStartIndexConstraints = {},
) => {
  let minDuration = Infinity;
  let minDurationIndex = 0;
  if (windowSize > maxValue) {
    windowSize = 0;
  }
  // If no constraints are specified, return the first value since it minimizes the number of 
  // iterations at the next higher level.
  if (!wrap && !windowSize) {
    return 0;
  }
  // From each value, calculate the forward duration to all other values within the constraints
  for (const [valueIndex, value] of sortedValues.entries()) {
    // If there are not enough values after this one, stop
    if (!wrap && valueIndex + windowSize >= sortedValues.length) {
      break;
    }
    // Calculate the duration to all values after this one
    let duration = 0;
    const limit = windowSize || (sortedValues.length - 1);
    for (let otherIndex = 0; otherIndex < limit; otherIndex++) {
      const other = sortedValues[(valueIndex + 1 + otherIndex) % sortedValues.length];
      // If the other value is after the value, calculate the duration
      if (other > value) {
        duration += other - value;
      } else if (wrap) {
        // The other value is before the value; calculate the duration after wrapping around
        duration += maxValue - value + 1 + other;
      } else {
        break;
      }
    }
    if (duration < minDuration) {
      minDuration = duration;
      minDurationIndex = valueIndex;
    }
  }
  return minDurationIndex;
};

const monthDays = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];

/** 
 * Generates days that are enabled by the cron expression.
 *
 * Yields increasing numbers representing any day enabled by either `days` or `weekdays`. The
 * numbers represent an absolute count of days since the start of the schedule. Follows calendar
 * months beginning in February.
 */
function* enabledCronDays(days: readonly number[], weekdays: readonly number[]) {
  let absoluteDay = 1;
  let monthDay = 1;
  let month = 1; // February
  const enabledDays = new Set(days.length === 31 && weekdays.length < 7 ? [] : days);
  const enabledWeekdays = new Set(weekdays.length === 7 ? [] : weekdays);
  while (true) {
    if (monthDay > monthDays[month]) {
      // Jump to the next month
      monthDay = 1;
      month = (month + 1) % 12;
    }
    if (enabledWeekdays.has(absoluteDay % 7) || enabledDays.has(monthDay)) {
      yield absoluteDay;
    }
    absoluteDay++;
    monthDay++;
  }
}

const hasConsecutiveValues = (sortedValues: readonly number[]) => sortedValues.some(
  (v, i) => i > 0 && v === sortedValues[i - 1] + 1,
);

// min/max range for each cron unit in the order returned by `converter.parseCronString`
const rangeByUnit = [
  [0, 59],
  [0, 23],
  [1, 31],
  [1, 12],
  [0, 6],
];

/** Parses a cron expression and returns the enabled values for each unit. */
const parseCronExpression = (cronExpr: string) => {
  const [ minutes, hours, days, months, weekdays ] = converter.parseCronString(cronExpr).map(
    (value, unitIndex) => (value.length > 0
      ? value
      // `parseCronString` returns `*` as an empty array; swap with the full range of values
      : Array.from(
        { length: rangeByUnit[unitIndex][1] - rangeByUnit[unitIndex][0] + 1 }, 
        (_, i) => rangeByUnit[unitIndex][0] + i
      )
    )
  );
  return { minutes, hours, days, months, weekdays };
};

/**
 * Calculates a rough minimum duration for the cron expression to run the specified number of times.
 * 
 * Months are ignored since backrest does not support scheduling by month.
 * @param cronExpr The cron expression to evaluate
 * @param targetInvocations The number of times the cron expression must run
 * @returns Minimal duration in milliseconds
 */
export const getMinimumCronDuration = (cronExpr: string, targetInvocations = 1) => {
  // Get arrays of minutes, hours, days, and weekdays that are enabled by the cron expression
  const { minutes, hours, days, weekdays } = parseCronExpression(cronExpr);
  // The total duration in minutes
  let elapsedMinutes = 0;
  let remainingInvocations = targetInvocations;
  const daysIterator = enabledCronDays(days, weekdays);
  // Start with just enough days to test whether there are consecutive days
  const daysAndWeekdays = Array.from(
    {
      length: weekdays.length < 7
        // Days and weekdays align sufficiently after about 100 invocations to test consecutive
        // days; no need to generate the full 7 month repeating pattern
        ? 100
        // If no weekdays specified we just need the days pattern with enough days to test wrapping
        : days.length * 2
    },
    () => +daysIterator.next().value,
  );
  let dayIndex = -1;
  do {
    // Calculate more days as needed
    if (dayIndex >= daysAndWeekdays.length) {
      daysAndWeekdays.push(+daysIterator.next().value);
    }

    // Add time elapsed between days
    if (dayIndex >= 0) {
      const skippedDays = (daysAndWeekdays[dayIndex] - (daysAndWeekdays[dayIndex - 1] ?? 0)) - 1;
      elapsedMinutes += skippedDays * 24 * 60;
    }

    let hourIndex = 0;
    do {
      const isFirstHour = elapsedMinutes === 0;
      if (!isFirstHour) {
        const skippedHours = hours[hourIndex] - (hours[hourIndex - 1] ?? 0) - 1;
        elapsedMinutes += skippedHours * 60;
      }

      let minuteStartIndex = 0;
      let minuteStartValue = 0;
      // Find the optimal start index for minutes on the first invocation
      if (isFirstHour) {
        const hasConsecutiveHours = hasConsecutiveValues(hours);
        minuteStartIndex = findMinimalDurationStartIndex(minutes, 59, {
          wrap: hasConsecutiveHours,
          windowSize: remainingInvocations,
        });
        minuteStartValue = minutes[minuteStartIndex];
      }

      // Need more invocations, so add all of the minute schedules
      if (remainingInvocations > minutes.length - minuteStartIndex) {
        elapsedMinutes += 60 - minuteStartValue;
        remainingInvocations -= minutes.length - minuteStartIndex;
      } else {
        // This iteration satisfies the remaining invocations, so add what's left
        elapsedMinutes += minutes[minuteStartIndex + remainingInvocations - 1] - minuteStartValue;
        // Return the duration in milliseconds
        return elapsedMinutes * 60_000;
      }

      // Need to keep iterating, so let's figure out which hour to start at
      if (isFirstHour) {
        const hasConsecutiveDays = hasConsecutiveValues(daysAndWeekdays);
        hourIndex = findMinimalDurationStartIndex(hours, 23, {
          wrap: hasConsecutiveDays,
          windowSize: Math.ceil(remainingInvocations / minutes.length),
        });
      }
      hourIndex++;
    } while (hourIndex < hours.length);

    // Add time elapsed through the end of the day
    const addHours = 24 - hours[hours.length - 1];
    elapsedMinutes += addHours * 60;

    if (dayIndex < 0) {
      // No need to calculate the start index until we know that more than one day is
      // needed
      dayIndex = findMinimalDurationStartIndex(
        daysAndWeekdays,
        daysAndWeekdays[daysAndWeekdays.length - 1],
        {
          // We're not actually moving through months, daysAndWeekdays is just an increasing 
          // sequence of day numbers
          wrap: false,
          windowSize: Math.ceil(remainingInvocations / hours.length / minutes.length),
        },
      );
    }
    dayIndex++;
  } while (true);
};