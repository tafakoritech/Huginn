import { useRef, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { PageHeader } from '@/components/ui/misc'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/Icon'
import { Input, Select, Textarea } from '@/components/ui/Field'
import { Progress } from '@/components/ui/misc'
import { toFaDigits } from '@/utils/format'
import { useAuth, PERMISSIONS } from '@/auth/AuthContext'

const schema = z.object({
  name: z.string().min(2),
  description: z.string().optional(),
  source_type: z.string().min(1),
  tags: z.string().optional(),
  project_id: z.string().optional(),
})

type FormValues = z.infer<typeof schema>

const SOURCE_TYPES = ['CSV', 'XLSX', 'PDF', 'DOCX', 'TXT', 'HTML', 'IMAGE', 'RAW_TEXT', 'URL', 'JSON']

export function ImportNewPage() {
  const { t } = useI18n()
  const { can } = useAuth()
  const toast = useToastApi()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const presetProject = searchParams.get('project') ?? ''
  const [files, setFiles] = useState<File[]>([])
  const [pasteText, setPasteText] = useState('')
  const [dragOver, setDragOver] = useState(false)
  const [uploadProgress, setUploadProgress] = useState<{ current: number; total: number } | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const canManageProjects = can(PERMISSIONS.projectsManage)
  const { data: projectsPage } = useQuery({
    queryKey: queryKeys.projects({ page_size: 100 }),
    queryFn: () => api.listProjects({ page_size: 100 }),
    enabled: canManageProjects,
  })
  const projectOptions = (projectsPage?.items ?? []).filter((p) => p.status === 'ACTIVE')

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: '', description: '', source_type: 'PDF', tags: '', project_id: presetProject },
  })

  const sourceType = watch('source_type')

  const createBatch = useMutation({
    mutationFn: (values: FormValues) =>
      api.createImportBatch({
        name: values.name,
        description: values.description,
        source_type: values.source_type,
        tags: values.tags ? values.tags.split(',').map((s) => s.trim()).filter(Boolean) : undefined,
        project_id: values.project_id || null,
      }),
    onSuccess: async (batch) => {
      const allFiles = [...files]
      if (pasteText.trim()) {
        allFiles.push(new File([pasteText], 'raw-text.txt', { type: 'text/plain' }))
      }
      if (allFiles.length > 0) {
        setUploadProgress({ current: 0, total: allFiles.length })
        // upload files sequentially to show honest progress
        const batchSize = 5
        for (let i = 0; i < allFiles.length; i += batchSize) {
          const chunk = allFiles.slice(i, i + batchSize)
          await api.uploadFiles(batch.id, chunk)
          setUploadProgress({ current: Math.min(i + chunk.length, allFiles.length), total: allFiles.length })
        }
        toast.success(t('imports.createSuccess'))
        toast.info(t('imports.processingNote'))
      } else {
        toast.success(t('imports.createSuccess'))
      }
      navigate(`/imports/${batch.id}`)
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const onDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    const dropped = Array.from(e.dataTransfer.files ?? [])
    if (dropped.length) setFiles((prev) => [...prev, ...dropped])
  }

  const addFiles = (list: FileList | null) => {
    if (list) setFiles((prev) => [...prev, ...Array.from(list)])
  }

  const onSubmit = (values: FormValues) => {
    if (files.length === 0 && !pasteText.trim()) {
      toast.warning(t('imports.dropzoneHint'))
    }
    createBatch.mutate(values)
  }

  return (
    <>
      <PageHeader title={t('imports.newImport')} subtitle={t('imports.subtitle')} />

      <div className="card" style={{ maxWidth: 860 }}>
        <form onSubmit={handleSubmit(onSubmit)} noValidate>
          <div className="card-body">
            <div className="form-row mb-4">
              <Input label={t('imports.batchName')} required placeholder={t('imports.batchName')} error={errors.name?.message} {...register('name')} />
              <Select label={t('imports.sourceType')} required {...register('source_type')}>
                {SOURCE_TYPES.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </Select>
            </div>
            <div className="mb-4">
              <Textarea label={t('imports.description')} placeholder={t('imports.description')} {...register('description')} style={{ minHeight: 70 }} />
            </div>
            <div className="mb-4">
              <Input label={t('imports.tags')} hint={t('common.optionalTag')} placeholder="فوری, منطقه مرکزی" {...register('tags')} />
            </div>
            {canManageProjects ? (
              <div className="mb-4">
                <Select label={t('projects.title')} hint={t('projects.assignLead')} {...register('project_id')}>
                  <option value="">{t('common.none')}</option>
                  {projectOptions.map((p) => (
                    <option key={p.id} value={p.id}>{p.title}</option>
                  ))}
                </Select>
              </div>
            ) : null}

            <div
              className={`dropzone ${dragOver ? 'dragging' : ''}`}
              onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
              onDragLeave={() => setDragOver(false)}
              onDrop={onDrop}
              onClick={() => fileInputRef.current?.click()}
              role="button"
              tabIndex={0}
              aria-label={t('imports.dropzone')}
            >
              <input
                ref={fileInputRef}
                type="file"
                multiple
                accept=".pdf,.xlsx,.xls,.csv,.docx,.txt,.html,.htm,.json,.png,.jpg,.jpeg"
                style={{ display: 'none' }}
                onChange={(e) => addFiles(e.target.files)}
              />
              <Icon name="upload" size={38} />
              <p className="strong">{t('imports.dropzone')}</p>
              <p className="small muted">{t('imports.dropzoneHint')}</p>
            </div>

            {files.length > 0 ? (
              <div className="flex flex-col gap-2 mt-4">
                {files.map((f, i) => (
                  <div className="upload-item" key={`${f.name}-${i}`}>
                    <Icon name="fileText" size={18} style={{ color: 'var(--text-3)' }} />
                    <span className="flex-1 ellipsis" dir="ltr" style={{ textAlign: 'start' }}>
                      {f.name}
                    </span>
                    <span className="small muted">{formatSize(f.size)}</span>
                    <button type="button" className="topbar-icon-btn" onClick={() => setFiles((prev) => prev.filter((_, j) => j !== i))} aria-label={t('common.remove')}>
                      <Icon name="x" size={15} />
                    </button>
                  </div>
                ))}
              </div>
            ) : null}

            <div className="flex items-center gap-3 my-4">
              <span className="muted small">{t('imports.or')}</span>
            </div>

            <div className="mb-4">
              <Textarea label={t('imports.pasteText')} placeholder={t('imports.pastePlaceholder')} value={pasteText} onChange={(e) => setPasteText(e.target.value)} style={{ minHeight: 100 }} />
            </div>

            {uploadProgress ? (
              <div className="mb-4">
                <div className="flex justify-between small secondary mb-2">
                  <span>{t('imports.uploadProgress')}</span>
                  <span>
                    {toFaDigits(uploadProgress.current)} / {toFaDigits(uploadProgress.total)}
                  </span>
                </div>
                <Progress value={(uploadProgress.current / Math.max(uploadProgress.total, 1)) * 100} />
              </div>
            ) : null}
          </div>

          <div className="card-footer flex justify-between items-center">
            <span className="small muted">{sourceType === 'RAW_TEXT' ? '' : t('imports.processingNote')}</span>
            <Button type="submit" variant="primary" icon="upload" loading={createBatch.isPending}>
              {t('imports.newImport')}
            </Button>
          </div>
        </form>
      </div>
    </>
  )
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
