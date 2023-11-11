/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

/**
 * base64 encoder and decoder
 * Copied and adapted from https://github.com/protobufjs/protobuf.js/blob/master/lib/base64/index.js
 */
// Base64 encoding table
const b64 = new Array(64);

// Base64 decoding table
const s64 = new Array(123);

// 65..90, 97..122, 48..57, 43, 47
for (let i = 0; i < 64;)
    s64[b64[i] = i < 26 ? i + 65 : i < 52 ? i + 71 : i < 62 ? i - 4 : i - 59 | 43] = i++;

export function b64Encode(buffer: Uint8Array, start: number, end: number): string {
	let parts: string[] = null;
  const chunk = [];
  let i = 0, // output index
    j = 0, // goto index
    t;     // temporary
  while (start < end) {
    const b = buffer[start++];
    switch (j) {
      case 0:
        chunk[i++] = b64[b >> 2];
        t = (b & 3) << 4;
        j = 1;
        break;
      case 1:
        chunk[i++] = b64[t | b >> 4];
        t = (b & 15) << 2;
        j = 2;
        break;
      case 2:
        chunk[i++] = b64[t | b >> 6];
        chunk[i++] = b64[b & 63];
        j = 0;
        break;
    }
    if (i > 8191) {
      (parts || (parts = [])).push(String.fromCharCode.apply(String, chunk));
      i = 0;
    }
  }
  if (j) {
    chunk[i++] = b64[t];
    chunk[i++] = 61;
    if (j === 1)
      chunk[i++] = 61;
  }
  if (parts) {
    if (i)
      parts.push(String.fromCharCode.apply(String, chunk.slice(0, i)));
    return parts.join("");
  }
  return String.fromCharCode.apply(String, chunk.slice(0, i));
}

const invalidEncoding = "invalid encoding";

export function b64Decode(s: string): Uint8Array {
	const buffer = [];
	let offset = 0;
  let j = 0, // goto index
      t;     // temporary
  for (let i = 0; i < s.length;) {
    let c = s.charCodeAt(i++);
    if (c === 61 && j > 1)
        break;
    if ((c = s64[c]) === undefined)
        throw Error(invalidEncoding);
    switch (j) {
      case 0:
        t = c;
        j = 1;
        break;
      case 1:
        buffer[offset++] = t << 2 | (c & 48) >> 4;
        t = c;
        j = 2;
        break;
      case 2:
        buffer[offset++] = (t & 15) << 4 | (c & 60) >> 2;
        t = c;
        j = 3;
        break;
      case 3:
        buffer[offset++] = (t & 3) << 6 | c;
        j = 0;
        break;
    }
  }
  if (j === 1)
      throw Error(invalidEncoding);
  return new Uint8Array(buffer);
}

function b64Test(s: string): boolean {
	return /^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(s);
}

export interface InitReq extends RequestInit {
  pathPrefix?: string
}

export function replacer(key: any, value: any): any {
  if(value && value.constructor === Uint8Array) {
    return b64Encode(value, 0, value.length);
  }

  return value;
}

export function fetchReq<I, O>(path: string, init?: InitReq): Promise<O> {
  const {pathPrefix, ...req} = init || {}

  const url = pathPrefix ? `${pathPrefix}${path}` : path

  return fetch(url, req).then(r => r.json().then((body: O) => {
    if (!r.ok) { throw body; }
    return body;
  })) as Promise<O>
}

// NotifyStreamEntityArrival is a callback that will be called on streaming entity arrival
export type NotifyStreamEntityArrival<T> = (resp: T) => void

/**
 * fetchStreamingRequest is able to handle grpc-gateway server side streaming call
 * it takes NotifyStreamEntityArrival that lets users respond to entity arrival during the call
 * all entities will be returned as an array after the call finishes.
 **/
export async function fetchStreamingRequest<S, R>(path: string, callback?: NotifyStreamEntityArrival<R>, init?: InitReq) {
  const {pathPrefix, ...req} = init || {}
  const url = pathPrefix ?`${pathPrefix}${path}` : path
  const result = await fetch(url, req)
  // needs to use the .ok to check the status of HTTP status code
  // http other than 200 will not throw an error, instead the .ok will become false.
  // see https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch#
  if (!result.ok) {
    const resp = await result.json()
    const errMsg = resp.error && resp.error.message ? resp.error.message : ""
    throw new Error(errMsg)
  }

  if (!result.body) {
    throw new Error("response doesnt have a body")
  }

  await result.body
    .pipeThrough(new TextDecoderStream())
    .pipeThrough<R>(getNewLineDelimitedJSONDecodingStream<R>())
    .pipeTo(getNotifyEntityArrivalSink((e: R) => {
      if (callback) {
        callback(e)
      }
    }))

  // wait for the streaming to finish and return the success respond
  return
}

/**
 * JSONStringStreamController represents the transform controller that's able to transform the incoming
 * new line delimited json content stream into entities and able to push the entity to the down stream
 */
interface JSONStringStreamController<T> extends TransformStreamDefaultController {
  buf?: string
  pos?: number
  enqueue: (s: T) => void
}

/**
 * getNewLineDelimitedJSONDecodingStream returns a TransformStream that's able to handle new line delimited json stream content into parsed entities
 */
function getNewLineDelimitedJSONDecodingStream<T>(): TransformStream<string, T> {
  return new TransformStream({
    start(controller: JSONStringStreamController<T>) {
      controller.buf = ''
      controller.pos = 0
    },

    transform(chunk: string, controller: JSONStringStreamController<T>) {
      if (controller.buf === undefined) {
        controller.buf = ''
      }
      if (controller.pos === undefined) {
        controller.pos = 0
      }
      controller.buf += chunk
      while (controller.pos < controller.buf.length) {
        if (controller.buf[controller.pos] === '\n') {
          const line = controller.buf.substring(0, controller.pos)
          const response = JSON.parse(line)
          controller.enqueue(response.result)
          controller.buf = controller.buf.substring(controller.pos + 1)
          controller.pos = 0
        } else {
          ++controller.pos
        }
      }
    }
  })

}

/**
 * getNotifyEntityArrivalSink takes the NotifyStreamEntityArrival callback and return
 * a sink that will call the callback on entity arrival
 * @param notifyCallback
 */
function getNotifyEntityArrivalSink<T>(notifyCallback: NotifyStreamEntityArrival<T>) {
  return new WritableStream<T>({
    write(entity: T) {
      notifyCallback(entity)
    }
  })
}

type Primitive = string | boolean | number;
type RequestPayload = Record<string, unknown>;
type FlattenedRequestPayload = Record<string, Primitive | Array<Primitive>>;

/**
 * Checks if given value is a plain object
 * Logic copied and adapted from below source: 
 * https://github.com/char0n/ramda-adjunct/blob/master/src/isPlainObj.js
 * @param  {unknown} value
 * @return {boolean}
 */
function isPlainObject(value: unknown): boolean {
  const isObject =
    Object.prototype.toString.call(value).slice(8, -1) === "Object";
  const isObjLike = value !== null && isObject;

  if (!isObjLike || !isObject) {
    return false;
  }

  const proto = Object.getPrototypeOf(value);

  const hasObjectConstructor =
    typeof proto === "object" &&
    proto.constructor === Object.prototype.constructor;

  return hasObjectConstructor;
}

/**
 * Checks if given value is of a primitive type
 * @param  {unknown} value
 * @return {boolean}
 */
function isPrimitive(value: unknown): boolean {
  return ["string", "number", "boolean"].some(t => typeof value === t);
}

/**
 * Checks if given primitive is zero-value
 * @param  {Primitive} value
 * @return {boolean}
 */
function isZeroValuePrimitive(value: Primitive): boolean {
  return value === false || value === 0 || value === "";
}

/**
 * Flattens a deeply nested request payload and returns an object
 * with only primitive values and non-empty array of primitive values
 * as per https://github.com/googleapis/googleapis/blob/master/google/api/http.proto
 * @param  {RequestPayload} requestPayload
 * @param  {String} path
 * @return {FlattenedRequestPayload>}
 */
function flattenRequestPayload<T extends RequestPayload>(
  requestPayload: T,
  path: string = ""
): FlattenedRequestPayload {
  return Object.keys(requestPayload).reduce(
    (acc: T, key: string): T => {
      const value = requestPayload[key];
      const newPath = path ? [path, key].join(".") : key;

      const isNonEmptyPrimitiveArray =
        Array.isArray(value) &&
        value.every(v => isPrimitive(v)) &&
        value.length > 0;

      const isNonZeroValuePrimitive =
        isPrimitive(value) && !isZeroValuePrimitive(value as Primitive);

      let objectToMerge = {};

      if (isPlainObject(value)) {
        objectToMerge = flattenRequestPayload(value as RequestPayload, newPath);
      } else if (isNonZeroValuePrimitive || isNonEmptyPrimitiveArray) {
        objectToMerge = { [newPath]: value };
      }

      return { ...acc, ...objectToMerge };
    },
    {} as T
  ) as FlattenedRequestPayload;
}

/**
 * Renders a deeply nested request payload into a string of URL search
 * parameters by first flattening the request payload and then removing keys
 * which are already present in the URL path.
 * @param  {RequestPayload} requestPayload
 * @param  {string[]} urlPathParams
 * @return {string}
 */
export function renderURLSearchParams<T extends RequestPayload>(
  requestPayload: T,
  urlPathParams: string[] = []
): string {
  const flattenedRequestPayload = flattenRequestPayload(requestPayload);

  const urlSearchParams = Object.keys(flattenedRequestPayload).reduce(
    (acc: string[][], key: string): string[][] => {
      // key should not be present in the url path as a parameter
      const value = flattenedRequestPayload[key];
      if (urlPathParams.find(f => f === key)) {
        return acc;
      }
      return Array.isArray(value)
        ? [...acc, ...value.map(m => [key, m.toString()])]
        : (acc = [...acc, [key, value.toString()]]);
    },
    [] as string[][]
  );

  return new URLSearchParams(urlSearchParams).toString();
}