export function notify(message, type = "info") {
  const el = document.createElement("div")
  el.className = `toast ${type}`
  el.textContent = message
  document.body.appendChild(el)
  setTimeout(() => el.remove(), 3000)
}