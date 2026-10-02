import { z } from 'zod'

import "@/models/items";
import "@/services/index.ts";

const noContentSchema = z.null()

function itemPath(id: number, suffix = ''): string {
  return `/api/items/${encodeURIComponent(String(id))}${suffix}`
}

function versionPath(id: number, suffix = ''): string {
  return `/api/versions/${encodeURIComponent(String(id))}${suffix}`
}

export function listItems(): Promise<ApiResult<Item[]>> {

}

export function listComponents(): Promise<ApiResult<Component[]>> {

}

export function listAssemblies(): Promise<ApiResult<Assembly[]>> {

}

export function getItem(id: number): Promise<ApiResult<Item>> {

}

export function getComponent(id: number): Promise<ApiResult<Component>> {

}

export function getAssembly(id: number): Promise<ApiResult<Assembly>> {

}

export function createComponent(params: CreateComponentParams): Promise<ApiResult<Component>> {

}

export function createAssembly(params: CreateAssemblyParams): Promise<ApiResult<CreateAssemblyResult>> {

}

export function listAssemblyVersions(assemblyId: number): Promise<ApiResult<BaseItemVersion[]>> {

}

export function getAssemblyVersion(versionId: number): Promise<ApiResult<ItemVersion>> {

}

export function createAssemblyVersion(assemblyId: number, params: CreateItemVersionParams): Promise<ApiResult<ItemVersion>> {

}

export function getBillOfMaterials(versionId: number): Promise<ApiResult<AssemblyNode>> {

}

export function publishVersion(versionId: number): Promise<ApiResult<null>> {

}

export function deprecateVersion(versionId: number): Promise<ApiResult<null>> {

}
