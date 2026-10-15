import { BASE_API_URL } from '@/services/index'

/**
 * The API serves an item's thumbnail as an image response. There is no
 * dedicated service helper for it, so components point an <img> at this URL.
 */
export function itemThumbnailUrl(itemId: number): string {
  return `${BASE_API_URL}/api/items/${itemId}/thumbnail`
}
