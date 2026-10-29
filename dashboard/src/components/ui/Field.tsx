import { forwardRef, useId, type InputHTMLAttributes, type SelectHTMLAttributes, type TextareaHTMLAttributes } from 'react'

type BaseProps = { label?: string; hint?: string; error?: string; required?: boolean }

export const Field = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement> & BaseProps>(
  function Field({ label, hint, error, required, className, id, ...rest }, ref) {
    const autoId = useId()
    const inputId = id ?? autoId
    return (
      <div className="field">
        {label ? (
          <label className="field-label" htmlFor={inputId}>
            {label} {required ? <span className="field-error">*</span> : null}
          </label>
        ) : null}
        <input ref={ref} id={inputId} className={`input ${error ? 'has-error' : ''} ${className ?? ''}`.trim()} aria-invalid={!!error} {...rest} />
        {hint ? <span className="field-hint">{hint}</span> : null}
        {error ? <span className="field-error" role="alert">{error}</span> : null}
      </div>
    )
  },
)

export const Select = forwardRef<HTMLSelectElement, SelectHTMLAttributes<HTMLSelectElement> & BaseProps>(
  function Select({ label, hint, error, required, className, id, children, ...rest }, ref) {
    const autoId = useId()
    const inputId = id ?? autoId
    return (
      <div className="field">
        {label ? (
          <label className="field-label" htmlFor={inputId}>
            {label} {required ? <span className="field-error">*</span> : null}
          </label>
        ) : null}
        <select ref={ref} id={inputId} className={`select ${error ? 'has-error' : ''} ${className ?? ''}`.trim()} aria-invalid={!!error} {...rest}>
          {children}
        </select>
        {hint ? <span className="field-hint">{hint}</span> : null}
        {error ? <span className="field-error" role="alert">{error}</span> : null}
      </div>
    )
  },
)

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaHTMLAttributes<HTMLTextAreaElement> & BaseProps>(
  function Textarea({ label, hint, error, required, className, id, ...rest }, ref) {
    const autoId = useId()
    const inputId = id ?? autoId
    return (
      <div className="field">
        {label ? (
          <label className="field-label" htmlFor={inputId}>
            {label} {required ? <span className="field-error">*</span> : null}
          </label>
        ) : null}
        <textarea ref={ref} id={inputId} className={`textarea ${error ? 'has-error' : ''} ${className ?? ''}`.trim()} aria-invalid={!!error} {...rest} />
        {hint ? <span className="field-hint">{hint}</span> : null}
        {error ? <span className="field-error" role="alert">{error}</span> : null}
      </div>
    )
  },
)

export function Checkbox({
  label,
  checked,
  onChange,
  disabled,
}: {
  label: string
  checked: boolean
  onChange: (v: boolean) => void
  disabled?: boolean
}) {
  return (
    <label className="checkbox">
      <input type="checkbox" checked={checked} disabled={disabled} onChange={(e) => onChange(e.target.checked)} />
      <span>{label}</span>
    </label>
  )
}

export { Field as Input }
