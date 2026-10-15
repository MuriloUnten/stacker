import type { Item } from '@/models/items'
import type { BomChildDraft } from '@/utils/bom'
import { emptyDraft } from '@/utils/bom'
import './BomLinesEditor.css'

interface BomLinesEditorProps {
  items: Item[]
  value: BomChildDraft[]
  disabled?: boolean
  onChange: (value: BomChildDraft[]) => void
}

/**
 * Editors for the BOM lines of an assembly. Only components and assemblies
 * that already have a published version can be picked as children, matching
 * the backend's validation rules.
 */
export function BomLinesEditor({
  items,
  value,
  disabled = false,
  onChange,
}: BomLinesEditorProps) {
  const selectable = items.filter(
    (item) => item.kind === 'c' || item.current_version !== null,
  )

  const updateDraft = (index: number, patch: Partial<BomChildDraft>) => {
    onChange(value.map((draft, i) => (i === index ? { ...draft, ...patch } : draft)))
  }

  const removeDraft = (index: number) => {
    onChange(value.filter((_, i) => i !== index))
  }

  return (
    <div className="bom-lines-editor">
      <div className="bom-lines-editor__head">
        <span className="field__label">children</span>
        <button
          type="button"
          onClick={() => onChange([...value, emptyDraft()])}
          disabled={disabled}
        >
          add child
        </button>
      </div>

      {value.length === 0 ? (
        <p className="bom-lines-editor__empty state">no children added</p>
      ) : (
        <ul className="bom-lines-editor__rows">
          {value.map((draft, index) => (
            <li className="bom-lines-editor__row" key={index}>
              <select
                value={draft.item_id === '' ? '' : String(draft.item_id)}
                onChange={(event) =>
                  updateDraft(index, {
                    item_id:
                      event.target.value === '' ? '' : Number(event.target.value),
                  })
                }
                disabled={disabled}
              >
                <option value="">select item…</option>
                {selectable.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name} · {item.sku}
                    {item.kind === 'a' && item.current_version
                      ? ` · v${item.current_version.version_code}`
                      : ''}
                  </option>
                ))}
              </select>

              <input
                className="bom-lines-editor__qty"
                type="number"
                min="0"
                step="any"
                value={draft.quantity}
                onChange={(event) => updateDraft(index, { quantity: event.target.value })}
                disabled={disabled}
              />

              <button
                type="button"
                onClick={() => removeDraft(index)}
                disabled={disabled}
              >
                remove
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
