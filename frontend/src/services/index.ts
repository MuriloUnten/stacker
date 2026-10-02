export const BASE_API_URL: string = import.meta.env.VITE_API_URL ?? "http://localhost:1337";

export interface PaginationParams {
  page?: number,
  pageSize?: number,
  limit?: boolean,
  paginate?: boolean,
}

export type SortOrder = "ascend" | "descend";

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const Methods = {
  GET: "GET",
  POST: "POST",
  PUT: "PUT",
  PATCH: "PATCH",
  DELETE: "DELETE",
} as const;

export type HTTPMethod = (typeof Methods)[keyof typeof Methods];

export type ApiResult<T> = ApiSuccess<T> | ApiError;

export type ApiSuccess<T> = {
  data: T;
  statusCode: number;
  success: true;
  error?: never;
};

export type ApiError = {
  success: false;
  statusCode?: number;
  error: Error;
};

export type HTTPResponse = {
  statusCode: number;
  body: object;
};

type RawHTTPResponse = {
  statusCode: number;
  headers: Headers;
  body: Blob;
}

class FetchError extends Error {
  constructor(message?: string) {
    super(message);
  }
}

class JsonParseError extends Error {
  constructor(message?: string) {
    super(message);
  }
}

type RequestResult = RequestSuccess | RequestError;
type RawRequestResult = RawRequestSuccess | RequestError;
type RawRequestSuccess = { ok: true, response: RawHTTPResponse }
type RequestSuccess = { ok: true, response: HTTPResponse };
// NOTE: This error refers to when something throws
// (4xx and 5xx status codes ARE NOT RequestErrors)
type RequestError = { ok: false, error: FetchError | JsonParseError };

type RequestOptions = {
  method?: HTTPMethod,
  headers?: Record<string, string>,
  body?: object,
  queryParams?: object,
}

/**
 * Makes a request to the api at <BASE_API_URL>/api.
 * Wraps a fetch() call with basic required extra authentication related content,
 * meaning it handles authentication transparently.
 *
 * @param path - Path of the request after the base url (leading slash '/' is optional). Query parameters must be passed via the `queryParams` object
 * @param options - Extra request options (method defaults to GET)
 * @returns response's body parsed as JSON
 */
export async function safeFetch(path: string, options: RequestOptions = {}): Promise<RequestResult> {
  const {
    method = "GET",
      headers,
    body,
    queryParams,
  } = options;

  const fullPath = getFullPath(path, queryParams);
  let formattedBody = "";
  const baseHeaders = {
    "Content-Type": "application/json",
  };

  try {
    formattedBody = JSON.stringify(body);
  } catch(e) {
    console.error(`Error parsing HTTP request body: ${e}`);
    return { ok: false, error: new JsonParseError() };
  }

  // make api call
  try {
    const response = await fetch(
      fullPath,
      {
        method: method,
        credentials: "include",
        headers: { ...baseHeaders, ...headers },
        body: formattedBody,
      }
    );

    // parse response body
    try {
      const data = await response.json();
      return {
        ok: true,
        response: {
          statusCode: response.status,
          body: data,
        },
      };
    } catch (e) {
      console.error(`Error parsing HTTP response: ${e}`);
      return { ok: false, error: new JsonParseError() };
    }
  } catch (e) {
    console.error(`Error in HTTP request: ${e}`);
    return { ok: false, error: new FetchError() };
  }
}

/**
 * Makes an api call and handles session state.
 * This is the one function that should be used for regular api calls.
 *
 * @param path    - Path of the request after the base url (leading slash '/' is optional). Query parameters must be passed via the `queryParams` object
 * @param options - Extra request options (method defaults to GET)
 * @returns response's body parsed as JSON
 */
export async function fetchApi(
  path: string,
  options: RequestOptions = {}
): Promise<RequestResult> {
    const result = await safeFetch(path, options);

    if (!result.ok) {
      return { ok: false, error: result.error };
    }

    return { ok: true, response: result.response };

    /* TODO add when implementing authentication */
    // // properly authorized, just return response
    // if (result.response.statusCode !== 401) {
    //   return { ok: true, response: result.response };
    // }
    //
    // // unauthorized, try to refresh tokens first
    // // triggerLogout=true because at this point the user had an active session that expired mid-use
    // const refreshResult = await auth.refresh(true);
    //
    // // in this particular case, a 401 is also considered an error and the session will be terminated
    // if (!refreshResult.success) {
    //   return { ok: false, error: refreshResult.error };
    // }
    //
    // // tokens refreshed, retry fetching
    // const retryResult = await safeFetch(path, options);
    // if (!retryResult.ok) {
    //   return { ok: false, error: retryResult.error };
    // }
    //
    // if (retryResult.response.statusCode === 401) {
    //   auth.triggerForceLogout();
    //   return { ok: false, error: new Error("Failed request retry after refreshing tokens") };
    // }
    //
    // return { ok: true, response: retryResult.response };
}

function getFullPath(path: string, queryParams?: object): string {
  let pathLocation = removeLeadingSlash(path);
  pathLocation = guaranteeTrailingSlash(pathLocation);

  let fullPath = `${BASE_API_URL}/api/${pathLocation}`;

  if (queryParams) {
    // iterate over query params and append them to the path
    fullPath += `?`
    Object.entries(queryParams).forEach(([key, value]) => {
      if (value === undefined || value === null) {
        return;
      }
      // NOTE: This does not account for values that don't stringify properly.
      // Should work for basic types and arrays for now.
      // TODO: Refactor this to handle more complex value objects
      fullPath += `${key}=${value}&`;
    });

    // delete last '&' character left over
    fullPath = fullPath.slice(0, -1);
  }

  return fullPath;
}

function removeLeadingSlash(str: string): string {
  if (!str.startsWith("/")) {
    return str;
  }

  return str.slice(1);
}

function guaranteeTrailingSlash(str: string): string {
  if (str.endsWith("/")) {
    return str;
  }

  return str + "/";
}

export function getOrdering(sortField?: string, sortOrder?: SortOrder): string | undefined {
  if (!sortField) return undefined;
  const direction = !sortOrder || sortOrder === "descend" ? "-" : "";
  return `${direction}${sortField}`;
}

function downloadFromResponse(response: RawHTTPResponse) {
  // Try to extract filename from header
  const contentDisposition = response.headers.get("Content-Disposition");
  let filename = "download";

  if (contentDisposition) {
    const match = contentDisposition.match(/filename="?(.+?)"?$/);
    if (match) filename = match[1];
  }

  const url = window.URL.createObjectURL(response.body);

  const a = document.createElement("a");
  a.href = url;
  a.download = filename;

  document.body.appendChild(a);
  a.click();

  a.remove();
  window.URL.revokeObjectURL(url);
}

export function downloadFile(filename: string, path: string) {
  const link = document.createElement("a");
  link.href = path;
  link.download = filename;
  document.body.appendChild(link);
  link.click();

  link.remove();
}
