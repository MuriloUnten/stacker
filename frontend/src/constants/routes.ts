export const ROUTES = {
    ROOT: "/",
    DASHBOARD: "/dashboard",
    ASSEMBLY: "/assemblies/:assemblyId",
    NOT_FOUND: "/404",
} as const;

export type AppRoute = (typeof ROUTES)[keyof typeof ROUTES];

export const ROUTE_PATHS = {
    ASSEMBLY: (assemblyId: string) => `/assemblies/${assemblyId}`,
} as const;
