import { z } from "zod";
import {
    assemblyListSchema,
    assemblyNodeSchema,
    assemblySchema,
    componentListSchema,
    componentSchema,
    createAssemblyResultSchema,
    itemListSchema,
    itemSchema,
    itemVersionListSchema,
    itemVersionSchema,
    type Assembly,
    type AssemblyNode,
    type BaseItemVersion,
    type Component,
    type CreateAssemblyParams,
    type CreateAssemblyResult,
    type CreateComponentParams,
    type CreateItemVersionParams,
    type Item,
    type ItemVersion,
} from "@/models/items";
import { serviceRequest, type ApiResult, serviceRequestWithoutResponseData } from '@/services/index';

export async function listItems(): Promise<ApiResult<Item[]>> {
    return await serviceRequest(itemListSchema, "/items");
}

export async function listComponents(): Promise<ApiResult<Component[]>> {
    return await serviceRequest(componentListSchema, "/components");
}

export async function listAssemblies(): Promise<ApiResult<Assembly[]>> {
    return await serviceRequest(assemblyListSchema, "/assemblies");
}

export async function getItem(id: number): Promise<ApiResult<Item>> {
    return await serviceRequest(itemSchema, `/items/${id}`);
}

export async function getComponent(id: number): Promise<ApiResult<Component>> {
    return await serviceRequest(componentSchema, `/components/${id}`);
}

export async function getAssembly(id: number): Promise<ApiResult<Assembly>> {
    return await serviceRequest(assemblySchema, `/assemblies/${id}`);
}

export async function createComponent(params: CreateComponentParams): Promise<ApiResult<Component>> {
    return await serviceRequest(
        componentSchema,
        "/components",
        { method: "POST", body: params },
    );
}

export async function createAssembly(params: CreateAssemblyParams): Promise<ApiResult<CreateAssemblyResult>> {
    return await serviceRequest(
        createAssemblyResultSchema,
        "/assemblies",
        { method: "POST", body: params },
    );
}

export async function listAssemblyVersions(assemblyId: number): Promise<ApiResult<BaseItemVersion[]>> {
    return await serviceRequest(itemVersionListSchema, `/assemblies/${assemblyId}/versions`);
}

export async function getAssemblyVersion(versionId: number): Promise<ApiResult<ItemVersion>> {
    return await serviceRequest(itemVersionSchema, `/versions/${versionId}`);
}

export async function createAssemblyVersion(assemblyId: number, params: CreateItemVersionParams): Promise<ApiResult<ItemVersion>> {
    return await serviceRequest(
        itemVersionSchema,
        `/assemblies/${assemblyId}/versions`,
        { method: "POST", body: params },
    );
}

export async function getBillOfMaterials(versionId: number): Promise<ApiResult<AssemblyNode>> {
    return await serviceRequest(assemblyNodeSchema, `/versions/${versionId}/bom`);
}

export async function publishVersion(versionId: number): Promise<ApiResult<null>> {
    return await serviceRequest(z.null(), `/versions/${versionId}/publish`, {
        method: "POST",
    });
}

export async function deprecateVersion(versionId: number): Promise<ApiResult<void>> {
    return await serviceRequestWithoutResponseData(
        `/versions/${versionId}/deprecate`,
        { method: "POST" }
    );
}
