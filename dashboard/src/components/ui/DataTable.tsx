import { type ReactNode } from 'react'
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
  type SortingState,
} from '@tanstack/react-table'
import { SortIndicator, Pagination } from './Pagination'
import { EmptyState } from './States'
import type { PaginationMeta } from '@/api/types'
import { useI18n } from '@/i18n'

export type DataTableProps<T> = {
  columns: ColumnDef<T, unknown>[]
  data: T[]
  meta?: PaginationMeta
  onPage?: (page: number) => void
  sorting?: SortingState
  onSortingChange?: (sorting: SortingState) => void
  loading?: boolean
  emptyTitle?: string
  emptyHint?: string
  emptyActions?: ReactNode
  rowSelection?: Record<string, boolean>
  onRowSelectionChange?: (sel: Record<string, boolean>) => void
  getRowId?: (row: T) => string
  onRowClick?: (row: T) => void
  footer?: ReactNode
  rowCountLabel?: string
}

export function DataTable<T>({
  columns,
  data,
  meta,
  onPage,
  sorting,
  onSortingChange,
  loading,
  emptyTitle,
  emptyHint,
  emptyActions,
  rowSelection,
  onRowSelectionChange,
  getRowId,
  onRowClick,
  footer,
  rowCountLabel,
}: DataTableProps<T>) {
  const { t } = useI18n()

  const table = useReactTable({
    data,
    columns,
    state: {
      sorting: sorting ?? [],
      rowSelection: rowSelection ?? {},
    },
    onSortingChange: (updater) => {
      if (!onSortingChange) return
      onSortingChange(typeof updater === 'function' ? updater(sorting ?? []) : updater)
    },
    onRowSelectionChange: (updater) => {
      if (!onRowSelectionChange) return
      onRowSelectionChange(typeof updater === 'function' ? updater(rowSelection ?? {}) : updater)
    },
    getRowId: getRowId ?? ((row) => (row as { id?: string }).id ?? String(Math.random())),
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    manualSorting: true,
  })

  const header = table.getHeaderGroups()[0]
  const rows = table.getRowModel().rows

  return (
    <div className="table-wrap">
      <div className="table-scroll">
        <table className="table">
          <thead>
            <tr>
              {header.headers.map((h) => {
                const sortable = h.column.getCanSort()
                const sorted = h.column.getIsSorted()
                return (
                  <th
                    key={h.id}
                    className={sortable ? 'sortable' : ''}
                    onClick={sortable && onSortingChange ? h.column.getToggleSortingHandler() : undefined}
                    aria-sort={sorted === 'asc' ? 'ascending' : sorted === 'desc' ? 'descending' : undefined}
                  >
                    <span className="flex items-center gap-1">
                      {flexRender(h.column.columnDef.header, h.getContext())}
                      {sortable ? <SortIndicator active={!!sorted} dir={sorted === 'asc' ? 'asc' : sorted === 'desc' ? 'desc' : null} /> : null}
                    </span>
                  </th>
                )
              })}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr aria-busy="true">
                <td colSpan={columns.length} style={{ padding: 16 }}>
                  <div className="flex flex-col gap-3">
                    {Array.from({ length: 5 }, (_, r) => (
                      <div className="flex gap-3" key={r}>
                        {header.headers.map((h) => (
                          <div className="skeleton skeleton-row flex-1" key={h.id} style={{ height: 16 }} />
                        ))}
                      </div>
                    ))}
                  </div>
                </td>
              </tr>
            ) : rows.length === 0 ? (
              <tr>
                <td colSpan={columns.length} className="table-empty">
                  <EmptyState title={emptyTitle ?? t('empty.noData')} hint={emptyHint} actions={emptyActions} />
                </td>
              </tr>
            ) : (
              rows.map((row) => {
                const selected = !!row.getIsSelected()
                return (
                  <tr
                    key={row.id}
                    className={selected ? 'selected' : ''}
                    onClick={onRowClick ? () => onRowClick(row.original) : undefined}
                    style={onRowClick ? { cursor: 'pointer' } : undefined}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <td key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</td>
                    ))}
                  </tr>
                )
              })
            )}
          </tbody>
        </table>
      </div>
      {meta && onPage ? (
        <div className="table-footer">
          {rowCountLabel ? <span className="small secondary">{rowCountLabel}</span> : <span />}
          <Pagination meta={meta} onPage={onPage} />
        </div>
      ) : footer ? (
        <div className="table-footer">{footer}</div>
      ) : null}
    </div>
  )
}
