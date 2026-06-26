import { describe, it, expect, vi, beforeEach } from 'vitest'
import { evaluateRuleWasm } from '../rules/wasmRuntime'

// Mock WebAssembly
const mockWebAssembly = {
  instantiateStreaming: vi.fn(),
}

// Mock global WebAssembly
Object.defineProperty(global, 'WebAssembly', {
  value: mockWebAssembly,
  writable: true,
})

// Mock document for dynamic script loading
const mockDocument = {
  createElement: vi.fn(),
  head: {
    appendChild: vi.fn(),
  },
}

Object.defineProperty(global, 'document', {
  value: mockDocument,
  writable: true,
})

// Mock fetch
const mockFetch = vi.fn()
Object.defineProperty(global, 'fetch', {
  value: mockFetch,
  writable: true,
})

describe('WASM Runtime', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Mock Go constructor
    ;(global as any).Go = vi.fn().mockImplementation(() => ({
      run: vi.fn(),
      importObject: {},
    }))
    // Mock evaluateRule function on window
    ;(global as any).evaluateRule = vi.fn()
  })

  it('should evaluate rule correctly', async () => {
    // Mock successful script loading
    const mockScript = {
      onload: null,
      onerror: null,
      src: '',
    }
    mockDocument.createElement.mockReturnValue(mockScript)

    // Mock successful fetch and WASM instantiation
    mockFetch.mockResolvedValue({
      arrayBuffer: vi.fn().mockResolvedValue(new ArrayBuffer(0)),
    })
    const mockInstance = {
      instance: {
        exports: {},
      },
    }
    mockWebAssembly.instantiateStreaming.mockResolvedValue(mockInstance)

    // Mock the evaluateRule function to return true
    ;(global as any).evaluateRule.mockReturnValue({ result: true })

    // Trigger script load
    setTimeout(() => {
      if (mockScript.onload) mockScript.onload(new Event('load'))
    }, 0)

    const rule = {
      type: 'Condition',
      condition: {
        field: 'status',
        operator: 'equals',
        value: 'active',
      },
    }
    const context = { status: 'active' }

    const result = await evaluateRuleWasm(rule, context)
    expect(result).toBe(true)
    expect((global as any).evaluateRule).toHaveBeenCalledWith(JSON.stringify(rule), JSON.stringify(context))
  })

  it('should handle script loading errors', async () => {
    // Remove evaluateRule from window to simulate failed initialization
    delete (global as any).evaluateRule;

    // Mock failed script loading
    const mockScript = {
      onload: null,
      onerror: null,
      src: '',
    }
    mockDocument.createElement.mockReturnValue(mockScript)

    // Trigger script error
    setTimeout(() => {
      if (mockScript.onerror) mockScript.onerror(new Event('error'))
    }, 0)

    const rule = {
      type: 'Condition',
      condition: {
        field: 'status',
        operator: 'equals',
        value: 'active',
      },
    }
    const context = { status: 'active' }

    await expect(evaluateRuleWasm(rule, context)).rejects.toThrow('WASM runtime not initialized')
  })

  it('should handle evaluation errors', async () => {
    // Mock successful script loading
    const mockScript = {
      onload: null,
      onerror: null,
      src: '',
    }
    mockDocument.createElement.mockReturnValue(mockScript)

    // Mock successful fetch and WASM instantiation
    mockFetch.mockResolvedValue({
      arrayBuffer: vi.fn().mockResolvedValue(new ArrayBuffer(0)),
    })
    const mockInstance = {
      instance: {
        exports: {},
      },
    }
    mockWebAssembly.instantiateStreaming.mockResolvedValue(mockInstance)

    // Mock the evaluateRule function to return an error
    ;(global as any).evaluateRule.mockReturnValue({ error: 'Evaluation failed' })

    // Trigger script load
    setTimeout(() => {
      if (mockScript.onload) mockScript.onload(new Event('load'))
    }, 0)

    const rule = {
      type: 'Condition',
      condition: {
        field: 'status',
        operator: 'equals',
        value: 'active',
      },
    }
    const context = { status: 'active' }

    await expect(evaluateRuleWasm(rule, context)).rejects.toThrow('Evaluation failed')
  })
})