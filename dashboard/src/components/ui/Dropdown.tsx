import { useState, type ReactNode } from 'react'
import { useClickOutside } from '@/hooks'
import { Icon } from '@/components/Icon'

export function Dropdown({
  trigger,
  children,
  align = 'end',
  className,
}: {
  trigger: ReactNode
  children: ReactNode
  align?: 'start' | 'end'
  className?: string
}) {
  const [open, setOpen] = useState(false)
  const ref = useClickOutside<HTMLDivElement>(() => setOpen(false))
  return (
    <div className="dropdown" ref={ref}>
      <div onClick={() => setOpen((o) => !o)}>{trigger}</div>
      {open ? (
        <div className={`menu ${align === 'start' ? 'menu-start' : ''} ${className ?? ''}`.trim()} role="menu">
          {children}
        </div>
      ) : null}
    </div>
  )
}

export function MenuItem({
  onClick,
  icon,
  children,
  danger,
}: {
  onClick?: () => void
  icon?: Parameters<typeof import('@/components/Icon').Icon>[0]['name']
  children: ReactNode
  danger?: boolean
}) {
  return (
    <button type="button" className={`menu-item ${danger ? 'danger' : ''}`.trim()} role="menuitem" onClick={onClick}>
      {icon ? <Icon name={icon} size={15} /> : null}
      {children}
    </button>
  )
}

export function MenuSeparator() {
  return <div className="menu-sep" role="separator" />
}
