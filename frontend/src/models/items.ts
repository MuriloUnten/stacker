import { z } from 'zod'

/**
 * The API uses shopspring/decimal values. They are emitted as JSON strings,
 * while accepting a number keeps the input models convenient for forms.
 */
const decimalPattern = /^-?(?:\d+(?:\.\d*)?|\.\d+)$/

const decimalSchema = z
  .union([z.string().regex(decimalPattern), z.number().finite()])
  .transform((value) => value.toString())

const dateTimeSchema = z
  .string()
  .refine((value) => !Number.isNaN(Date.parse(value)), 'Expected an ISO date-time')

const idSchema = z.number().int().positive()
const inventoryItemIdSchema = z.number().int().nonnegative()
const nullableIdSchema = idSchema.nullable()

export const itemKindSchema = z.enum(['c', 'a'])
export type ItemKind = z.infer<typeof itemKindSchema>

export const unitOfMeasurementSchema = z.enum([
  'each',
  'meter',
  'centimeter',
  // Keep the API's existing spelling so responses validate without a translation layer.
  'millimiter',
  'gram',
  'kilogram',
  'meter_sqr',
])
export type UnitOfMeasurement = z.infer<typeof unitOfMeasurementSchema>

export const bomStatusSchema = z.enum(['draft', 'published', 'deprecated'])
export type BomStatus = z.infer<typeof bomStatusSchema>

export const baseItemSchema = z.object({
  id: idSchema,
  name: z.string(),
  sku: z.string(),
  kind: itemKindSchema,
  description: z.string(),
  thumbnail_id: nullableIdSchema,
})

export type BaseItem = z.infer<typeof baseItemSchema>

export const inventorySchema = z.object({
  // The current API emits the inventory relation id as 0 for item responses.
  item_id: inventoryItemIdSchema,
  unit: unitOfMeasurementSchema,
  available: decimalSchema,
  reserved: decimalSchema,
})

export type Inventory = z.infer<typeof inventorySchema>

export const baseItemVersionSchema = z.object({
  id: idSchema,
  item_id: idSchema,
  version_code: z.string(),
  notes: z.string(),
  status: bomStatusSchema,
  created_at: dateTimeSchema,
  published_at: dateTimeSchema.nullable(),
})

export type BaseItemVersion = z.infer<typeof baseItemVersionSchema>

export const itemSchema = baseItemSchema.extend({
  inventory: inventorySchema,
  current_version: baseItemVersionSchema.nullable(),
})

export type Item = z.infer<typeof itemSchema>

export const componentSchema = itemSchema.extend({
  kind: z.literal('c'),
})

export type Component = z.infer<typeof componentSchema>

export const assemblySchema = itemSchema.extend({
  kind: z.literal('a'),
})

export type Assembly = z.infer<typeof assemblySchema>

export const bomLineSchema = z.object({
  id: idSchema,
  parent_version_id: idSchema,
  child_item_id: idSchema,
  child_version_id: nullableIdSchema,
  quantity: decimalSchema,
  position: z.number().int().nonnegative(),
})

export type BomLine = z.infer<typeof bomLineSchema>

export const itemVersionSchema = baseItemVersionSchema.extend({
  children: z.array(bomLineSchema),
})

export type ItemVersion = z.infer<typeof itemVersionSchema>

export interface AssemblyNode {
  id: number
  name: string
  sku: string
  kind: ItemKind
  description: string
  thumbnail_id: number | null
  inventory: Inventory
  version: BaseItemVersion | null
  quantity: string
  position: number
  // Go nil slices serialize as null, especially for component nodes.
  children: AssemblyNode[] | null
}

export const assemblyNodeSchema: z.ZodType<AssemblyNode> = z.lazy(() =>
  z.object({
    id: idSchema,
    name: z.string(),
    sku: z.string(),
    kind: itemKindSchema,
    description: z.string(),
    thumbnail_id: nullableIdSchema,
    inventory: inventorySchema,
    version: baseItemVersionSchema.nullable(),
    quantity: decimalSchema,
    position: z.number().int().nonnegative(),
    children: z.array(assemblyNodeSchema).nullable(),
  }),
)

export const createBaseItemParamsSchema = z.object({
  name: z.string().trim().min(1, 'Name is required'),
  sku: z.string().trim().min(1, 'SKU is required'),
  description: z.string().trim(),
  unit: unitOfMeasurementSchema,
  available: decimalSchema,
})

export const createComponentParamsSchema = createBaseItemParamsSchema
export type CreateComponentParams = z.infer<typeof createComponentParamsSchema>

export const createAssemblyChildParamsSchema = z.object({
  item_id: idSchema,
  item_version_id: nullableIdSchema,
  quantity: decimalSchema,
})

export type CreateAssemblyChildParams = z.infer<typeof createAssemblyChildParamsSchema>

export const createItemVersionParamsSchema = z.object({
  version_code: z.string().trim().min(1, 'Version code is required'),
  version_notes: z.string().trim(),
  children: z.array(createAssemblyChildParamsSchema).min(1, 'Add at least one child'),
})

export type CreateItemVersionParams = z.infer<typeof createItemVersionParamsSchema>

export const createAssemblyParamsSchema = createBaseItemParamsSchema.extend(
  createItemVersionParamsSchema.shape,
)

export type CreateAssemblyParams = z.infer<typeof createAssemblyParamsSchema>

export const createAssemblyResultSchema = assemblySchema.extend({
  version: itemVersionSchema,
})

export type CreateAssemblyResult = z.infer<typeof createAssemblyResultSchema>

export const itemListSchema = z.array(itemSchema)
export const componentListSchema = z.array(componentSchema)
export const assemblyListSchema = z.array(assemblySchema)
export const itemVersionListSchema = z.array(baseItemVersionSchema)
