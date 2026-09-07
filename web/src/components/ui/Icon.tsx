import type { LucideIcon } from 'lucide-react'

interface IconProps {
  icon: LucideIcon
  size?: number
  className?: string
  /** ตั้งชื่อเมื่อไอคอนยืนลำพังโดยไม่มีข้อความกำกับ ให้โปรแกรมอ่านหน้าจอเข้าใจ */
  label?: string
}

/**
 * ห่อ lucide-react ไว้ชั้นเดียว เพื่อคุมขนาดและความหนาเส้นให้เหมือนกันทั้งแอป
 * สีรับมาจาก currentColor เสมอ ไอคอนจึงเปลี่ยนตามธีมและตามสถานะของ element ที่ครอบอยู่เอง
 */
export function Icon({ icon: Lucide, size = 16, className, label }: IconProps) {
  return (
    <Lucide
      size={size}
      strokeWidth={1.75}
      className={className}
      aria-hidden={label ? undefined : true}
      aria-label={label}
      role={label ? 'img' : undefined}
    />
  )
}
