import {
  useLoaderData,
  useParams,
  type LoaderFunction,
  type RouteObject,
} from "react-router";
import { Home } from "./pages/home";
import type React from "react";
import { Search } from "./pages/search";
import { BookView } from "./pages/book";
import { bookList, bookView } from "./api/api";

const routes = {
  home: {
    path: "/",
    Component: Home as React.ComponentType,
  },
  "book.search": {
    path: "/book/search",
    Component: Search as React.ComponentType,
  },
  "book.view": {
    path: "/book/:bookId",
    loader: async (args) => {
      const book = await bookView(Number(args.params.bookId));
      return {
        book: book,
      };
    },
    Component: BookView as React.ComponentType,
  },
} as const satisfies Record<string, Omit<RouteObject, "id">>;

type RouteParts<
  T extends string,
  O = never,
> = T extends `/${infer First}/${infer Rest}`
  ? RouteParts<`/${Rest}`, O | First>
  : T extends `/${infer First}`
    ? O | First
    : O;

type OptionalArg<T extends string> = T extends `:${infer Key}?` ? Key : never;
type RequiredArg<T extends string> = T extends `:${infer _Key}?`
  ? never
  : T extends `:${infer Key}`
    ? Key
    : never;

type RouteParams<T extends string> = {
  [P in RequiredArg<RouteParts<T>>]: string | number;
} & {
  [P in OptionalArg<RouteParts<T>>]?: string | number;
};

export type Routes = typeof routes;
export type RouteName = keyof Routes;
export type Route = Routes[RouteName];

export type RouteParams2<T extends RouteName> = RouteParams<Routes[T]["path"]>;

export function buildRoutes(): RouteObject[] {
  return Object.entries(routes).map(([id, route]) => ({
    ...route,
    id: id,
  }));
}

type RouteArgs<T extends RouteName> = keyof RouteParams<
  Routes[T]["path"]
> extends never
  ? [name: T, params?: RouteParams<Routes[T]["path"]>]
  : [name: T, params: RouteParams<Routes[T]["path"]>];

export function routePath<T extends RouteName>(...args: RouteArgs<T>): string;
export function routePath<T extends RouteName>(
  name: T,
  args?: RouteParams<Routes[T]["path"]>,
): string {
  const route = routes[name];
  let path: string = route.path;
  for (const [key, value] of Object.entries(args ?? {})) {
    path = path.replace(`:${key}`, encodeURIComponent(String(value)));
  }
  path = path.replace(/\/:[^/]+\?/, "");
  return path;
}

export function useRouteParams<T extends RouteName>(): RouteParams<
  Routes[T]["path"]
> {
  return useParams() as any;
}

type ReturnType<T extends (...args: any) => any> = T extends (
  ...args: any
) => infer R
  ? R
  : any;

interface WithLoader {
  loader: LoaderFunction;
}

type RouteLoaderData<T extends RouteName> = Awaited<
  ReturnType<Routes[T] extends WithLoader ? Routes[T]["loader"] : never>
>;

export function useRouteLoaderData<T extends RouteName>(): RouteLoaderData<T> {
  return useLoaderData() as any;
}
