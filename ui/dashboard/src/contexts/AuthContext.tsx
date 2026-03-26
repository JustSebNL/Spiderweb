import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { toast } from 'react-hot-toast'

interface AuthState {
  isAuthenticated: boolean
  user: User | null
  isLoading: boolean
}

interface User {
  id: string
  username: string
  email: string
  roles: string[]
  permissions: string[]
}

interface AuthContextType {
  state: AuthState
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  refreshToken: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export const useAuth = () => {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

interface AuthProviderProps {
  children: ReactNode
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [state, setState] = useState<AuthState>({
    isAuthenticated: false,
    user: null,
    isLoading: true,
  })

  useEffect(() => {
    checkAuth()
  }, [])

  const checkAuth = async () => {
    try {
      const token = localStorage.getItem('auth_token')
      if (token) {
        // Verify token with backend
        const response = await fetch('/api/v1/auth/verify', {
          headers: {
            'Authorization': `Bearer ${token}`,
          },
        })
        
        if (response.ok) {
          const userData = await response.json()
          setState({
            isAuthenticated: true,
            user: userData.user,
            isLoading: false,
          })
        } else {
          localStorage.removeItem('auth_token')
          setState({
            isAuthenticated: false,
            user: null,
            isLoading: false,
          })
        }
      } else {
        setState({
          isAuthenticated: false,
          user: null,
          isLoading: false,
        })
      }
    } catch (error) {
      console.error('Auth check failed:', error)
      setState({
        isAuthenticated: false,
        user: null,
        isLoading: false,
      })
    }
  }

  const login = async (username: string, password: string) => {
    try {
      const response = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username, password }),
      })

      if (!response.ok) {
        throw new Error('Login failed')
      }

      const data = await response.json()
      localStorage.setItem('auth_token', data.token)
      
      setState({
        isAuthenticated: true,
        user: data.user,
        isLoading: false,
      })

      toast.success('Welcome back!')
    } catch (error) {
      toast.error('Login failed. Please check your credentials.')
      throw error
    }
  }

  const logout = () => {
    localStorage.removeItem('auth_token')
    setState({
      isAuthenticated: false,
      user: null,
      isLoading: false,
    })
    toast.success('Logged out successfully')
  }

  const refreshToken = async () => {
    try {
      const token = localStorage.getItem('auth_token')
      if (!token) return

      const response = await fetch('/api/v1/auth/refresh', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })

      if (response.ok) {
        const data = await response.json()
        localStorage.setItem('auth_token', data.token)
      } else {
        logout()
      }
    } catch (error) {
      console.error('Token refresh failed:', error)
      logout()
    }
  }

  const value: AuthContextType = {
    state,
    login,
    logout,
    refreshToken,
  }

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  )
}