import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Routes, Navigate, Route } from 'react-router'
import './index.css'
import { ROUTES } from './constants/routes'
import { Dashboard } from './pages/dashboard/Dashboard'
import { Assemblies } from './pages/assemblies/Assemblies'
import { Assembly } from './pages/assembly/Assembly'
import { NotFound } from './pages/not_found/404'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path={ROUTES.ROOT} element={<Navigate to={ROUTES.DASHBOARD} replace />} />
        <Route path={ROUTES.DASHBOARD} element={<Dashboard />} />
        <Route path={ROUTES.ASSEMBLIES} element={<Assemblies />} />
        <Route path={ROUTES.ASSEMBLY} element={<Assembly />} />
        <Route path={ROUTES.NOT_FOUND} element={<NotFound />} />
        <Route path="*" element={<Navigate to={ROUTES.NOT_FOUND} replace />} />
      </Routes>
    </BrowserRouter>
  </StrictMode>,
)
