import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router'
import './index.css'
import App from './App.tsx'

// Without transitions, a navigation renders in the same pass as the state changes made alongside it.
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter useTransitions={false}>
      <App />
    </BrowserRouter>
  </StrictMode>,
)
