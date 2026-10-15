import { Link } from 'react-router'
import type { Item, ItemKind } from '@/models/items'
import { ROUTE_PATHS } from '@/constants/routes'
import { itemThumbnailUrl } from '@/utils/thumbnail'
import './ItemCard.css'

const KIND_LABELS: Record<ItemKind, string> = {
  a: 'assembly',
  c: 'component',
}

interface ItemCardProps {
  item: Item
}

export function ItemCard({ item }: ItemCardProps) {
  const body = (
    <>
      <div className="item-card__thumb">
        {item.thumbnail_id === null ? (
          <span className="item-card__thumb-empty">no image</span>
        ) : (
          <img src={itemThumbnailUrl(item.id)} alt={item.name} loading="lazy" />
        )}
      </div>
      <div className="item-card__info">
        <div className="item-card__top">
          <h2 className="item-card__name">{item.name}</h2>
          <span className={`item-card__kind item-card__kind--${item.kind}`}>
            {KIND_LABELS[item.kind]}
          </span>
        </div>
        <p className="item-card__sku">{item.sku}</p>
        {item.description ? (
          <p className="item-card__description">{item.description}</p>
        ) : null}
        <dl className="item-card__meta">
          <div>
            <dt>stock</dt>
            <dd>
              {item.inventory.available} {item.inventory.unit}
            </dd>
          </div>
          <div>
            <dt>reserved</dt>
            <dd>{item.inventory.reserved}</dd>
          </div>
          <div>
            <dt>version</dt>
            <dd>
              {item.current_version
                ? `${item.current_version.version_code} (${item.current_version.status})`
                : '—'}
            </dd>
          </div>
        </dl>
      </div>
    </>
  )

  if (item.kind === 'a') {
    return (
      <Link className="item-card item-card--link" to={ROUTE_PATHS.ASSEMBLY(String(item.id))}>
        {body}
      </Link>
    )
  }

  return <div className="item-card">{body}</div>
}
