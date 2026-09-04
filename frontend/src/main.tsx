import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { BrowserRouter, Route, Routes } from 'react-router-dom'

createRoot(document.getElementById('root')!).render(
  <BrowserRouter>
    <Routes>
      <Route path='/' element={<App />} />
      <Route path='/login' />
      <Route path='/login/validation' />
      <Route path='/register' />
      <Route path='/register/validation' />
      <Route path='/forgetPassword' />
      <Route path='/forgetPassword/validate' />
      <Route path='/forgetPassword/:key' />
      
    </Routes>
  </BrowserRouter>
)
