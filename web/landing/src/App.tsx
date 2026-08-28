import { BrowserRouter } from 'react-router-dom'
import { AppRoutes } from '@/router'

export function App() {
  return (
    <BrowserRouter>
      <AppRoutes />
    </BrowserRouter>
  )
}
