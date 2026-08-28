import { Navigate, Route, Routes } from 'react-router-dom'
import { LangLayout } from '@/components/layout/LangLayout'

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/:lang/*" element={<LangLayout />} />
      <Route index element={<Navigate to="/cn" replace />} />
      <Route path="*" element={<Navigate to="/cn" replace />} />
    </Routes>
  )
}
