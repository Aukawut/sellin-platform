import { useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import type { MapNode } from '../../types/api'
import { LAKE_PATH, PROVINCES, THAILAND_VIEWBOX } from './thailand-provinces.generated'
import { DC_COLORS, REGION_COLOR, dcAnchor, provinceIdsFor } from './thailand'
import { deltaText, thbShort } from '../../lib/format'

interface Marker {
  code: string
  name: string
  x: number
  y: number
  r: number
  color: string
  node: MapNode
}

interface CanvasProps {
  markers: Marker[]
  /** รหัสจังหวัดที่ต้องระบายสีเน้น พร้อมสีที่ใช้ */
  shaded?: Map<string, string>
  selected?: string
  showPct?: boolean
  tooltip: (m: Marker) => ReactNode
  height?: string
}

/**
 * แผนที่ประเทศไทยระดับจังหวัด วาดจากขอบเขตจริงของทั้ง 77 จังหวัด
 * ที่มาข้อมูล: @svg-maps/thailand (CC-BY-4.0)
 */
function MapCanvas({ markers, shaded, selected, showPct, tooltip, height = 'max-h-[460px]' }: CanvasProps) {
  const [hover, setHover] = useState<Marker | null>(null)

  return (
    <div className="relative">
      <svg
        viewBox={THAILAND_VIEWBOX}
        className={`mx-auto block h-auto w-full ${height}`}
        role="img"
        aria-label="แผนที่ประเทศไทยแสดงตำแหน่งศูนย์กระจายสินค้า"
      >
        <g>
          {PROVINCES.map((p) => {
            const tint = shaded?.get(p.id)
            return (
              <path
                key={p.id}
                d={p.d}
                fill={tint ?? 'var(--sunk)'}
                fillOpacity={tint ? 0.3 : 1}
                stroke="var(--line)"
                strokeWidth={0.6}
              >
                <title>{p.th}</title>
              </path>
            )
          })}
        </g>

        {LAKE_PATH && <path d={LAKE_PATH} fill="var(--ground)" stroke="var(--line)" strokeWidth={0.6} />}

        {markers.map((m) => {
          const dimmed = selected && selected !== m.code
          return (
            <g
              key={m.code}
              opacity={dimmed ? 0.25 : m.node.value === 0 && showPct ? 0.4 : 1}
              onMouseEnter={() => setHover(m)}
              onMouseLeave={() => setHover(null)}
              style={{ cursor: 'pointer' }}
            >
              <circle cx={m.x} cy={m.y} r={m.r + 6} fill={m.color} opacity={0.2} />
              <circle cx={m.x} cy={m.y} r={m.r} fill={m.color} stroke="var(--paper)" strokeWidth={2.5} />
              {showPct && (
                <text
                  x={m.x}
                  y={m.y + 5}
                  textAnchor="middle"
                  fontSize={14}
                  fontWeight={700}
                  fill="#fff"
                  style={{ pointerEvents: 'none' }}
                >
                  {m.node.pct.toFixed(0)}%
                </text>
              )}
            </g>
          )
        })}
      </svg>

      <p className="mt-1 text-center text-[10px] text-faint">
        ขอบเขตจังหวัดจาก @svg-maps/thailand · CC BY 4.0
      </p>

      {hover && (
        <div
          className="pointer-events-none absolute z-10 max-w-[250px] rounded-lg border border-line bg-paper px-3 py-2 text-[12px] shadow-panel"
          style={{
            left: `min(${(hover.x / 560) * 100}%, calc(100% - 260px))`,
            top: `max(${(hover.y / 1025) * 100}% - 14px, 0px)`,
          }}
        >
          {tooltip(hover)}
        </div>
      )}
    </div>
  )
}

/** แผนที่เครือข่ายศูนย์กระจายสินค้า — หมุดขนาดเท่ากันเพราะสื่อว่า "มีศูนย์ดูแลพื้นที่นี้" */
export function NetworkMap({ nodes, selected }: { nodes: MapNode[]; selected?: string }) {
  const { markers, shaded } = useMemo(() => {
    const markers: Marker[] = []
    const shaded = new Map<string, string>()

    nodes.forEach((n, i) => {
      const color = DC_COLORS[i % DC_COLORS.length]
      for (const id of provinceIdsFor(n.provinces)) shaded.set(id, color)

      const anchor = dcAnchor(n.provinces)
      if (anchor) {
        markers.push({ code: n.code, name: n.name, x: anchor[0], y: anchor[1], r: 14, color, node: n })
      }
    })
    return { markers, shaded }
  }, [nodes])

  return (
    <MapCanvas
      markers={markers}
      shaded={shaded}
      selected={selected}
      tooltip={(m) => (
        <>
          <p className="font-semibold">{m.name}</p>
          <p className="mt-0.5 text-muted">
            ดูแล {m.node.provinces?.length ?? 0} จังหวัด
            {m.node.provinces?.length ? ` · ${m.node.provinces.join(', ')}` : ''}
          </p>
        </>
      )}
    />
  )
}

/** แผนที่สัดส่วนรายภาค — ขนาดหมุดแปรตามยอดขาย จึงเทียบภาคต่อภาคได้จากภาพเดียว */
export function RegionMap({
  nodes, customers,
}: {
  nodes: MapNode[]
  customers: { code: string; region?: string; provinces?: string[] }[]
}) {
  const { markers, shaded } = useMemo(() => {
    const max = Math.max(0, ...nodes.map((n) => n.value))
    const shaded = new Map<string, string>()

    // ระบายสีทุกจังหวัดตามภาคที่ศูนย์ในภาคนั้นดูแล
    for (const c of customers) {
      const color = c.region ? REGION_COLOR[c.region] : undefined
      if (!color) continue
      for (const id of provinceIdsFor(c.provinces)) shaded.set(id, color)
    }

    const markers = nodes
      .map((n) => {
        // จุดกึ่งกลางของภาค คือค่าเฉลี่ยของจังหวัดที่ศูนย์ในภาคนั้นดูแลอยู่จริง
        const provinces = customers.filter((c) => c.region === n.code).flatMap((c) => c.provinces ?? [])
        const anchor = dcAnchor(provinces)
        if (!anchor) return null
        return {
          code: n.code,
          name: `ภาค${n.name}`,
          x: anchor[0],
          y: anchor[1],
          r: n.value > 0 && max > 0 ? 14 + 20 * (n.value / max) : 10,
          color: REGION_COLOR[n.code] ?? 'var(--faint)',
          node: n,
        } satisfies Marker
      })
      .filter((m): m is Marker => m !== null)

    return { markers, shaded }
  }, [nodes, customers])

  return (
    <MapCanvas
      markers={markers}
      shaded={shaded}
      showPct
      tooltip={(m) => (
        <>
          <p className="font-semibold">{m.name}</p>
          <p className="tnum mt-0.5">
            Sell-Out {thbShort(m.node.value)} · {m.node.pct.toFixed(1)}% ของยอดรวม
          </p>
          <p className="mt-0.5 text-muted">
            {m.node.yoy ? deltaText(m.node.yoy) : 'ไม่มีข้อมูลปีก่อนหน้าให้เทียบ'}
          </p>
        </>
      )}
    />
  )
}
