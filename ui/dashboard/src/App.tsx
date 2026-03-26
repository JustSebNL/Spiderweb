import React from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from './contexts/AuthContext'
import { Layout } from './components/Layout'
import { Login } from './pages/Login'
import { Dashboard } from './pages/Dashboard'
import { Agents } from './pages/Agents'
import { Tasks } from './pages/Tasks'
import { Events } from './pages/Events'
import { Configuration } from './pages/Configuration'
import { LoadingSpinner } from './components/LoadingSpinner'

function App() {
  const { state } = useAuth()

  if (state.isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    )
  }

  if (!state.isAuthenticated) {
    return <Login />
  }

  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/agents" element={<Agents />} />
        <Route path="/tasks" element={<Tasks />} />
        <Route path="/events" element={<Events />} />
        <Route path="/configuration" element={<Configuration />} />
      </Routes>
    </Layout>
  )
}

export default App