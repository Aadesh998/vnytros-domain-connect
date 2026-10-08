import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import { App } from './App.tsx'
import { ConfigError } from './components/ConfigError.tsx'
import { isConfigured } from './lib/config'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    {isConfigured ? <App /> : <ConfigError />}
  </StrictMode>,
)
