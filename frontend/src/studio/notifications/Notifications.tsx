import React, { useState, useEffect } from 'react'

export function Notifications() {
  const [items, setItems] = useState([])

  useEffect(() => {
    window.notify = (msg, type = "info") => {
      const id = Math.random().toString()
      setItems(items => [...items, { id, msg, type }])
      setTimeout(() => {
        setItems(items => items.filter(i => i.id !== id))
      }, 3000)
    }
  }, [])

  return (
    <div className="notifications">
      {items.map(i => (
        <div key={i.id} className={`toast toast-${i.type}`}>
          {i.msg}
        </div>
      ))}
    </div>
  )
}