import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from './AppShell'
import { LoginPage } from '../features/auth/LoginPage'
import { DatasetLibrary } from '../features/datasets/DatasetLibrary'
import { SellInDashboard } from '../features/dashboard/SellInPage'
import { SellOutDashboard } from '../features/dashboard/SellOutPage'
import { StockDashboard } from '../features/dashboard/StockPage'
import { PlanningDashboard } from '../features/dashboard/PlanningPage'
import { AdminPage } from '../features/admin/AdminPage'
import { useAuth } from '../features/auth/AuthContext'
import { Spinner } from '../components/ui/States'

export function AppRoutes() {
  const { user, restoring } = useAuth()

  // ระหว่างกู้เซสชันยังไม่รู้ว่าล็อกอินอยู่หรือไม่
  // ถ้าเด้งไปหน้าล็อกอินทันที ผู้ใช้ที่ล็อกอินค้างไว้จะเห็นหน้าล็อกอินกระพริบทุกครั้งที่รีเฟรช
  if (restoring) return <Spinner label="กำลังตรวจสอบเซสชัน" />

  if (!user) {
    return (
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    )
  }

  return (
    <Routes>
      <Route path="/login" element={<Navigate to="/datasets" replace />} />
      <Route element={<AppShell />}>
        <Route path="/datasets" element={<DatasetLibrary />} />
        <Route path="/dashboard/sellin" element={<SellInDashboard />} />
        <Route path="/dashboard/sellout" element={<SellOutDashboard />} />
        <Route path="/dashboard/stock" element={<StockDashboard />} />
        <Route path="/dashboard/planning" element={<PlanningDashboard />} />
        <Route path="/settings" element={<AdminPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/datasets" replace />} />
    </Routes>
  )
}
