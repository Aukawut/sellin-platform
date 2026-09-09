import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  BarChart3, Check, Download, FileSpreadsheet, Info, RotateCcw,
  Trash2, TriangleAlert, XCircle,
} from 'lucide-react'
import { Button } from '../../components/ui/Button'
import { Icon } from '../../components/ui/Icon'
import { Pill } from '../../components/ui/Pill'
import { EmptyState, ErrorState, Spinner } from '../../components/ui/States'
import { UploadPanel } from './UploadPanel'
import { useDatasetDetail, useDatasets, useDeleteDataset, useRestoreDataset } from './api'
import { bytes, formatDate } from '../../lib/format'
import { downloadFromApi } from '../../lib/download'
import type { Dataset } from '../../types/api'

const SHEET_ORDER = [
  'Data_Sell-In', 'Data_Sell-Out', 'Data_Stock_inventory',
  'Target_Sale', 'Tracking_Non-Sun Flower Seed', 'Data_Product', 'Data_Customer',
]

export function DatasetLibrary() {
  const [showTrash, setShowTrash] = useState(false)
  const [watching, setWatching] = useState<string | null>(null)
  const datasets = useDatasets(showTrash)
  const remove = useDeleteDataset()
  const restore = useRestoreDataset()
  const navigate = useNavigate()

  // ติดตามสถานะเฉพาะไฟล์ที่เพิ่งอัปโหลด ไม่ต้อง poll ทั้งรายการ
  useDatasetDetail(watching, true)

  return (
    <div className="dataset-library mx-auto grid w-full max-w-[1180px] gap-6 px-5 py-7 lg:grid-cols-[minmax(0,1fr)_340px]">
      <section className="order-2 lg:order-1">
        <header className="mb-4 flex flex-wrap items-end justify-between gap-3">
          <div>
            <h1 className="font-display text-[28px] font-bold tracking-tight">คลังไฟล์ข้อมูล</h1>
            <p className="mt-0.5 text-[13px] text-muted">
              เลือกไฟล์ที่ต้องการ แล้วระบบจะสร้าง dashboard จากชุดข้อมูลนั้น
            </p>
          </div>
          <div className="flex items-center gap-0.5 rounded-lg border border-line bg-paper p-0.5">
            {[
              { v: false, label: 'ใช้งานอยู่' },
              { v: true, label: 'ถังขยะ' },
            ].map((t) => (
              <button
                key={String(t.v)}
                type="button"
                onClick={() => setShowTrash(t.v)}
                className={`rounded-[6px] px-3 py-1.5 text-[12.5px] font-semibold transition ${
                  showTrash === t.v ? 'bg-info-wash text-info' : 'text-muted hover:text-ink'
                }`}
              >
                {t.label}
              </button>
            ))}
          </div>
        </header>

        {datasets.isPending && <Spinner label="กำลังโหลดรายการไฟล์" />}
        {datasets.isError && (
          <ErrorState message="โหลดรายการไฟล์ไม่สำเร็จ" retry={() => datasets.refetch()} />
        )}

        {datasets.data?.length === 0 && (
          <EmptyState
            icon={showTrash ? Trash2 : FileSpreadsheet}
            title={showTrash ? 'ถังขยะว่างเปล่า' : 'ยังไม่มีไฟล์ในระบบ'}
            detail={
              showTrash
                ? 'ไฟล์ที่ลบจะมาอยู่ที่นี่และกู้คืนได้ ข้อมูลยังอยู่ครบไม่ได้ถูกลบจริง'
                : 'อัปโหลดไฟล์ Template ทางด้านขวาเพื่อเริ่มสร้าง dashboard'
            }
          />
        )}

        <ul className="flex flex-col gap-3">
          {datasets.data?.map((d) => (
            <DatasetCard
              key={d.id}
              dataset={d}
              inTrash={showTrash}
              busy={remove.isPending || restore.isPending}
              onOpen={() => navigate(`/dashboard/sellin?dataset=${d.id}`)}
              onDelete={() => remove.mutate(d.id)}
              onRestore={() => restore.mutate(d.id)}
            />
          ))}
        </ul>
      </section>

      <aside className="order-1 lg:order-2">
        <UploadPanel onUploaded={setWatching} />
      </aside>
    </div>
  )
}

function DatasetCard({
  dataset: d, inTrash, busy, onOpen, onDelete, onRestore,
}: {
  dataset: Dataset
  inTrash: boolean
  busy: boolean
  onOpen: () => void
  onDelete: () => void
  onRestore: () => void
}) {
  const detail = useDatasetDetail(d.id, d.status === 'processing')
  const live = detail.data?.dataset ?? d
  const issues = detail.data?.import.issue_counts

  const totalRows = Object.values(live.row_counts ?? {}).reduce((a, b) => a + b, 0)

  return (
    <li className="dataset-card rounded-2xl border border-line bg-paper p-5 shadow-panel">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <Icon icon={FileSpreadsheet} size={16} className="shrink-0 text-faint" />
            <p className="truncate font-display text-[14.5px] font-semibold">
              {live.label || live.original_filename}
            </p>
            <StatusPill status={live.status} />
          </div>
          <p className="mt-1 truncate text-[12.5px] text-muted">
            {live.label && <span>{live.original_filename} · </span>}
            {bytes(live.size_bytes)} · {live.owner_name} · {formatDate(live.uploaded_at)}
          </p>
        </div>

        <div className="flex shrink-0 items-center gap-2">
          {inTrash ? (
            <Button icon={RotateCcw} disabled={!live.can_delete || busy} onClick={onRestore}>
              กู้คืน
            </Button>
          ) : (
            <>
              <button
                type="button"
                onClick={() =>
                  void downloadFromApi(`/datasets/${live.id}/file`, live.original_filename)
                }
                className="grid size-9 place-items-center rounded-lg border border-line text-muted transition hover:bg-sunk hover:text-ink"
                title="ดาวน์โหลดไฟล์ต้นฉบับ"
              >
                <Icon icon={Download} size={15} label="ดาวน์โหลดไฟล์ต้นฉบับ" />
              </button>
              <button
                type="button"
                disabled={!live.can_delete || busy}
                onClick={onDelete}
                title={live.can_delete ? 'ลบ (กู้คืนได้)' : 'ลบได้เฉพาะไฟล์ของตัวเอง'}
                className="grid size-9 place-items-center rounded-lg border border-line text-muted transition hover:bg-bad-wash hover:text-bad disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-muted"
              >
                <Icon icon={Trash2} size={15} label="ลบไฟล์" />
              </button>
              <Button
                variant="primary"
                icon={BarChart3}
                disabled={live.status !== 'ready'}
                onClick={onOpen}
              >
                เปิด Dashboard
              </Button>
            </>
          )}
        </div>
      </div>

      {live.status === 'failed' && live.error_message && (
        <p className="mt-3 flex items-start gap-2 rounded-lg bg-bad-wash px-3 py-2 text-[12.5px] text-bad">
          <Icon icon={XCircle} size={14} className="mt-0.5 shrink-0" />
          {live.error_message}
        </p>
      )}

      {live.status === 'ready' && (
        <div className="mt-3 border-t border-line-soft pt-3">
          <div className="flex flex-wrap gap-x-5 gap-y-1.5 text-[12px]">
            {SHEET_ORDER.filter((s) => live.row_counts?.[s]).map((s) => (
              <span key={s} className="text-muted">
                {s} <span className="tnum font-semibold text-ink">{live.row_counts[s].toLocaleString()}</span>
              </span>
            ))}
          </div>
          <p className="mt-2 flex flex-wrap items-center gap-2 text-[12px] text-faint">
            <span className="tnum">รวม {totalRows.toLocaleString()} แถว</span>
            {live.year_min && (
              <span>· ปี {live.year_min === live.year_max ? live.year_min : `${live.year_min}–${live.year_max}`}</span>
            )}
            {issues && issues.warning + issues.blocking > 0 && (
              <Pill tone="warn">
                <span className="inline-flex items-center gap-1">
                  <Icon icon={TriangleAlert} size={11} />
                  {issues.blocking + issues.warning} ข้อควรตรวจ
                </span>
              </Pill>
            )}
            {issues && issues.warning + issues.blocking === 0 && (
              <Pill tone="good">
                <span className="inline-flex items-center gap-1">
                  <Icon icon={Check} size={11} />
                  ไม่พบปัญหา
                </span>
              </Pill>
            )}
            {issues && issues.info > 0 && (
              <span className="inline-flex items-center gap-1">
                <Icon icon={Info} size={11} />
                {issues.info} หมายเหตุ
              </span>
            )}
          </p>
        </div>
      )}
    </li>
  )
}

function StatusPill({ status }: { status: Dataset['status'] }) {
  if (status === 'processing') return <Pill tone="info">กำลังนำเข้า…</Pill>
  if (status === 'failed') return <Pill tone="bad">นำเข้าไม่สำเร็จ</Pill>
  return null
}
