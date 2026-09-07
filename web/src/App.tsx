import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthProvider } from './features/auth/AuthContext'
import { AppRoutes } from './app/routes'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // ตัวเลขในชุดข้อมูลไม่เปลี่ยนเองระหว่างที่เปิดหน้าอยู่ เพราะมันคือสแนปช็อตของไฟล์ที่อัปโหลดแล้ว
      // จึงไม่ต้องดึงซ้ำเมื่อสลับแท็บกลับมา
      staleTime: 5 * 60 * 1000,
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthProvider>
          <AppRoutes />
        </AuthProvider>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
