import * as React from "react"
import {
  ToastProvider,
  ToastViewport,
  Toast,
  ToastTitle,
  ToastDescription,
  ToastClose,
} from "./toast"
import { useToast as useAppToast, toast as baseToast } from "../../hooks/use-toast"

// Local Toaster component using Radix primitives (shadcn/ui style)
const Toaster: React.FC = () => {
  const { toasts } = useAppToast()
  return (
    <ToastProvider>
      {toasts.map(function ({ id, title, description, action, ...props }) {
        return (
          <Toast key={id} {...props}>
            <div className="grid gap-1">
              {title && <ToastTitle>{title}</ToastTitle>}
              {description && (
                <ToastDescription>{description}</ToastDescription>
              )}
            </div>
            {action}
            <ToastClose />
          </Toast>
        )
      })}
      <ToastViewport />
    </ToastProvider>
  )
}

// Shim a sonner-like API for convenience
type ToastInput =
  | string
  | {
      title?: string
      description?: string
      variant?: "default" | "destructive"
      [key: string]: any
    }

function toast(input: ToastInput) {
  if (typeof input === "string") {
    return baseToast({ description: input })
  }
  return baseToast(input)
}

toast.success = (message: string) =>
  baseToast({ title: "Success", description: message, variant: "default" })

toast.error = (message: string) =>
  baseToast({ title: "Error", description: message, variant: "destructive" })

export { Toaster, toast }
