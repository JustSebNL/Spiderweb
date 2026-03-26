import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react'
import { io, Socket } from 'socket.io-client'

interface SocketContextType {
  socket: Socket | null
  isConnected: boolean
  events: Event[]
  subscribeToEvents: (eventTypes: string[]) => void
  unsubscribeFromEvents: (eventTypes: string[]) => void
}

interface Event {
  id: string
  type: string
  timestamp: string
  source: string
  severity: 'info' | 'warning' | 'error' | 'success'
  message: string
  data?: any
}

const SocketContext = createContext<SocketContextType | undefined>(undefined)

export const useSocket = () => {
  const context = useContext(SocketContext)
  if (context === undefined) {
    throw new Error('useSocket must be used within a SocketProvider')
  }
  return context
}

interface SocketProviderProps {
  children: ReactNode
}

export const SocketProvider: React.FC<SocketProviderProps> = ({ children }) => {
  const [socket, setSocket] = useState<Socket | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [events, setEvents] = useState<Event[]>([])

  useEffect(() => {
    const token = localStorage.getItem('auth_token')
    if (!token) return

    const newSocket = io(import.meta.env.VITE_WS_URL || 'ws://localhost:8080', {
      auth: {
        token,
      },
    })

    newSocket.on('connect', () => {
      setIsConnected(true)
      console.log('WebSocket connected')
    })

    newSocket.on('disconnect', () => {
      setIsConnected(false)
      console.log('WebSocket disconnected')
    })

    newSocket.on('event', (event: Event) => {
      setEvents(prev => [event, ...prev.slice(0, 999)]) // Keep last 1000 events
    })

    setSocket(newSocket)

    return () => {
      newSocket.close()
    }
  }, [])

  const subscribeToEvents = (eventTypes: string[]) => {
    if (socket) {
      socket.emit('subscribe', { eventTypes })
    }
  }

  const unsubscribeFromEvents = (eventTypes: string[]) => {
    if (socket) {
      socket.emit('unsubscribe', { eventTypes })
    }
  }

  const value: SocketContextType = {
    socket,
    isConnected,
    events,
    subscribeToEvents,
    unsubscribeFromEvents,
  }

  return (
    <SocketContext.Provider value={value}>
      {children}
    </SocketContext.Provider>
  )
}