import React, { useState, useEffect } from 'react'

interface NotificationItem {
  id: string;
  msg: string;
  type: "success" | "info" | "warning" | "error";
}

export function Notifications() {
  const [items, setItems] = useState<NotificationItem[]>([])

  useEffect(() => {
    window.notify = (msg: string, type = "info") => {
      const id = Math.random().toString()
      setItems((items: NotificationItem[]) => [...items, { id, msg, type } as NotificationItem])
      setTimeout(() => {
        setItems((items: NotificationItem[]) => items.filter((i: NotificationItem) => i.id !== id))
      }, 3000)
    }
  }, [])

  return (
    <div className="notifications">
      {items.map((i: NotificationItem) => (
        <div key={i.id} className={`toast toast-${i.type}`}>
          {i.msg}
        </div>
      ))}
    </div>
  )
}