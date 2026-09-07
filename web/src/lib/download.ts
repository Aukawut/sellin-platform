import { ApiError, authHeader } from './api'

/**
 * ดาวน์โหลดไฟล์จาก API ที่ต้องแนบ token
 *
 * ใช้ลิงก์ <a href> ตรง ๆ ไม่ได้ เพราะ access token อยู่ในหน่วยความจำ ไม่ใช่ cookie
 * เบราว์เซอร์จึงไม่แนบไปให้ ต้องดึงด้วย fetch แล้วค่อยสร้างลิงก์ชั่วคราวจาก blob
 */
export async function downloadFromApi(path: string, fallbackName: string): Promise<void> {
  const res = await fetch(`/api/v1${path}`, {
    headers: authHeader(),
    credentials: 'include',
  })

  if (!res.ok) {
    let message = `ดาวน์โหลดไม่สำเร็จ (${res.status})`
    try {
      const data = await res.json()
      if (typeof data?.error === 'string') message = data.error
    } catch {
      /* เซิร์ฟเวอร์อาจไม่ได้ตอบเป็น JSON */
    }
    throw new ApiError(message, res.status)
  }

  const name = filenameFromHeader(res.headers.get('content-disposition')) ?? fallbackName
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)

  const a = document.createElement('a')
  a.href = url
  a.download = name
  document.body.appendChild(a)
  a.click()
  a.remove()

  // ปล่อย URL ทิ้งหลังเบราว์เซอร์เริ่มบันทึกไฟล์แล้ว ถ้าเพิกถอนทันทีบางเบราว์เซอร์จะได้ไฟล์เปล่า
  setTimeout(() => URL.revokeObjectURL(url), 10_000)
}

/** อ่านชื่อไฟล์จาก header โดยรองรับรูปแบบ filename*=UTF-8'' ที่ใช้กับชื่อภาษาไทย */
function filenameFromHeader(header: string | null): string | undefined {
  if (!header) return undefined

  const utf8 = header.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8) {
    try {
      return decodeURIComponent(utf8[1].trim())
    } catch {
      /* ชื่อไฟล์เข้ารหัสมาไม่ถูกต้อง ใช้ชื่อสำรองแทน */
    }
  }
  const plain = header.match(/filename="?([^";]+)"?/i)
  return plain ? plain[1].trim() : undefined
}
