
export function debounce<T extends (...args: any[]) => void>(
  func: T,
  wait: number,
  options: { maxWait?: number; trailing?: boolean } = {}
): T & { cancel: () => void } {
  let timeoutId: any | null = null;
  let maxWaitTimeoutId: any | null = null;
  let lastArgs: any[] | null = null;
  let lastThis: any | null = null;
  let lastCallTime: number | null = null;

  const invoke = () => {
    if (timeoutId) {
      clearTimeout(timeoutId);
      timeoutId = null;
    }
    if (maxWaitTimeoutId) {
      clearTimeout(maxWaitTimeoutId);
      maxWaitTimeoutId = null;
    }
    if (lastArgs) {
      func.apply(lastThis, lastArgs);
      lastArgs = null;
      lastThis = null;
    }
  };

  const debounced = function (this: any, ...args: any[]) {
    // eslint-disable-next-line @typescript-eslint/no-this-alias
    lastThis = this;
    lastArgs = args;
    lastCallTime = Date.now();

    if (timeoutId) {
      clearTimeout(timeoutId);
    }

    timeoutId = setTimeout(invoke, wait);

    if (options.maxWait && !maxWaitTimeoutId) {
      maxWaitTimeoutId = setTimeout(invoke, options.maxWait);
    }
  } as T & { cancel: () => void };

  debounced.cancel = () => {
    if (timeoutId) {
      clearTimeout(timeoutId);
      timeoutId = null;
    }
    if (maxWaitTimeoutId) {
      clearTimeout(maxWaitTimeoutId);
      maxWaitTimeoutId = null;
    }
  };

  return debounced;
}

export function groupBy<T>(
  array: T[],
  iteratee: (item: T) => string | number
): Record<string, T[]> {
  const result: Record<string, T[]> = {};
  for (const item of array) {
    const key = iteratee(item);
    if (!result[key]) {
      result[key] = [];
    }
    result[key].push(item);
  }
  return result;
}

export function keyBy<T>(
  array: T[],
  iteratee: (item: T) => string | number
): Record<string, T> {
  const result: Record<string, T> = {};
  for (const item of array) {
    const key = iteratee(item);
    result[key] = item;
  }
  return result;
}
