import type { CreateAssemblyChildParams, Item } from '@/models/items'

/**
 * Draft representation of a single BOM line while it is being edited in a form.
 * `item_id` is empty until the user picks an item.
 */
export interface BomChildDraft {
  item_id: number | ''
  quantity: string
}

export function emptyDraft(): BomChildDraft {
  return { item_id: '', quantity: '1' }
}

/**
 * Resolves edited drafts into the params the API expects. Components never
 * carry a version; assemblies use their currently published version.
 */
export function toAssemblyChildren(
  drafts: BomChildDraft[],
  items: Item[],
): CreateAssemblyChildParams[] {
  return drafts.map((draft) => {
    const item = items.find((candidate) => candidate.id === draft.item_id)
    const itemVersionId = item && item.kind === 'a' ? item.current_version?.id ?? null : null

    return {
      item_id: typeof draft.item_id === 'number' ? draft.item_id : 0,
      item_version_id: itemVersionId,
      quantity: draft.quantity,
    }
  })
}
