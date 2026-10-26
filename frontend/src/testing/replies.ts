import { http as msw, HttpResponse } from 'msw';
import type { HttpHandler } from 'msw';
import { createOpenApiHttp } from 'openapi-msw';
import type { PathsFor, ResponseBodyFor } from 'openapi-msw';
import type { components, paths } from '@/api/generated/schema';

export const ORIGIN = window.location.origin;

/** MSW's request handlers typed from the contract, for answers `reply` cannot give. */
export const http = createOpenApiHttp<paths>({ baseUrl: ORIGIN });

type Typed = typeof http;
const HANDLERS = {
  GET: msw.get,
  POST: msw.post,
  PUT: msw.put,
  PATCH: msw.patch,
  DELETE: msw.delete,
};
type Method = keyof typeof HANDLERS;
const METHODS: readonly Method[] = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];
interface Lower {
  GET: 'get';
  POST: 'post';
  PUT: 'put';
  PATCH: 'patch';
  DELETE: 'delete';
}
type PathFor<M extends Method> = PathsFor<Typed[Lower[M]]>;

/** An operation of the contract, written as "GET /api/v1/plans/{planId}". */
export type ApiKey = { [M in Method]: `${M} ${PathFor<M>}` }[Method];

/** Any body the operation `K` answers with, in any of its statuses. */
export type BodyOf<K extends ApiKey> = {
  [M in Method]: K extends `${M} ${infer P extends PathFor<M>}`
    ? ResponseBodyFor<Typed[Lower[M]], P>
    : never;
}[Method];

/** A canned answer: a status (200 when left out) and a body (none when left out). */
export interface Reply<K extends ApiKey> {
  status?: number;
  body?: BodyOf<K>;
}

/**
 * A canned answer, or a function for anything special: answers that change from call to call, a
 * Response that is not JSON, an answer that never comes, or a network failure (a rejection).
 */
export type Answer<K extends ApiKey> =
  Reply<K> | ((request: Request) => Reply<K> | Response | Promise<Reply<K> | Response>);

/**
 * Answers the operation `key` of the contract; with `params`, only for those path parameters.
 *
 *   reply('GET /api/v1/plans/{planId}', { body: plan }, { planId: 'P-1004' })
 */
export function reply<K extends ApiKey>(
  key: K,
  answer: Answer<K>,
  params?: Record<string, string>,
): HttpHandler {
  return looseReply(key, answer, params);
}

/** An answer whose body is not checked against the contract. */
export type LooseAnswer =
  | { status?: number; body?: unknown }
  | ((request: Request) => LooseAnswer | Response | Promise<LooseAnswer | Response>);

/**
 * `reply` without the contract check of the body: only for tests of a feature that is not aligned
 * yet (026); every feature moves to `reply` as it is aligned.
 */
export function looseReply(
  key: ApiKey,
  answer: LooseAnswer,
  params?: Record<string, string>,
): HttpHandler {
  const [method, path] = parseKey(key);
  return HANDLERS[method](
    ORIGIN + path.replace(/\{(\w+)\}/g, ':$1'),
    ({ request, params: actual }) =>
      !params || Object.entries(params).every(([name, value]) => actual[name] === value)
        ? respond(answer, request)
        : undefined,
  );
}

type ErrorDetail = components['schemas']['ErrorDetail'];

/** The API's error envelope, with no field problems unless `detail` gives them. */
export function apiError(
  code: ErrorDetail['code'],
  message: string,
  detail: Partial<ErrorDetail> = {},
): components['schemas']['Error'] {
  return { error: { code, message, fields: [], ...detail } };
}

/** An answer that never comes: the screen stays in its loading state. */
export function pending(): Promise<never> {
  return new Promise<never>(() => {});
}

async function respond(answer: LooseAnswer, request: Request): Promise<Response> {
  let result: LooseAnswer | Response;
  try {
    result = typeof answer === 'function' ? await answer(request) : answer;
  } catch {
    return HttpResponse.error();
  }
  if (result instanceof Response) {
    return result;
  }
  if (typeof result === 'function') {
    return respond(result, request);
  }
  const { status = 200, body } = result;
  return body === undefined
    ? new Response(null, { status })
    : new Response(JSON.stringify(body), {
        status,
        headers: { 'Content-Type': 'application/json' },
      });
}

function parseKey(key: string): [Method, string] {
  const [method = '', path = ''] = key.split(' ');
  const known = METHODS.find((name) => name === method);
  if (!known) {
    throw new Error(`Not an operation: ${key}`);
  }
  return [known, path];
}
