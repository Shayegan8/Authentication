import { createRoot } from 'react-dom/client'
import './index.css'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import App from './App'
import { RootElementContext } from './RootElementContext'

const rootDOMElement = document.getElementById('root') as HTMLDivElement

createRoot(rootDOMElement).render(
  <RootElementContext.Provider value={rootDOMElement!}>
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<App />} />
        <Route path="/register" element={<App />} />
        <Route path="/register/validation" element={<App />} />
        <Route path="/register/validation/jwt/:redirected" element={<App />} />
        <Route path="/register/validation/submit" element={<App />} />
        <Route path="/login" element={<App />} />
        <Route path="/login/validation" element={<App />} />
        <Route path="/login/validation/jwt" element={<App />} />
        <Route path="/login/validation/submit" element={<App />} />
        <Route path="/forgetPassword" element={<App />} />
        <Route path="/forgetPassword/validate" element={<App />} />
        <Route path="/forgetPassword/validate/jwt" element={<App />} />
        <Route path="/forgetPassword/:key" element={<App />} />
        <Route path="/posts" element={<App />} />
        <Route path="/post/:postid" element={<App />} />
        <Route path="/dashboard" element={<App />} />
        <Route path="*" element={<App />} />
      </Routes>
    </BrowserRouter>
  </RootElementContext.Provider>
)
