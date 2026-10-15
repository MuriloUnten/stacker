import { unitOfMeasurementSchema } from '@/models/items'
import type { UnitOfMeasurement } from '@/models/items'
import './BaseItemFields.css'

export interface BaseItemFieldsValue {
  name: string
  sku: string
  description: string
  unit: UnitOfMeasurement
  available: string
}

interface BaseItemFieldsProps {
  value: BaseItemFieldsValue
  disabled?: boolean
  onChange: (patch: Partial<BaseItemFieldsValue>) => void
}

export function BaseItemFields({
  value,
  disabled = false,
  onChange,
}: BaseItemFieldsProps) {
  return (
    <>
      <label className="field">
        <span className="field__label">name</span>
        <input
          value={value.name}
          onChange={(event) => onChange({ name: event.target.value })}
          disabled={disabled}
        />
      </label>

      <label className="field">
        <span className="field__label">sku</span>
        <input
          value={value.sku}
          onChange={(event) => onChange({ sku: event.target.value })}
          disabled={disabled}
        />
      </label>

      <label className="field">
        <span className="field__label">description</span>
        <textarea
          rows={2}
          value={value.description}
          onChange={(event) => onChange({ description: event.target.value })}
          disabled={disabled}
        />
      </label>

      <div className="base-item-fields__row">
        <label className="field">
          <span className="field__label">unit</span>
          <select
            value={value.unit}
            onChange={(event) =>
              onChange({ unit: event.target.value as UnitOfMeasurement })
            }
            disabled={disabled}
          >
            {unitOfMeasurementSchema.options.map((unit) => (
              <option key={unit} value={unit}>
                {unit}
              </option>
            ))}
          </select>
        </label>

        <label className="field">
          <span className="field__label">available</span>
          <input
            type="number"
            step="any"
            value={value.available}
            onChange={(event) => onChange({ available: event.target.value })}
            disabled={disabled}
          />
        </label>
      </div>
    </>
  )
}
