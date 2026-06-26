import { ipPatternsOverlap } from '../utils/ipValidation'

describe('ipPatternsOverlap', () => {
  test('overlaps when wildcard contains specific ip', () => {
    expect(ipPatternsOverlap('192.168.*.*', '192.168.1.1')).toBe(true)
  })

  test('does not overlap when different networks', () => {
    expect(ipPatternsOverlap('10.*.*.*', '11.*.*.*')).toBe(false)
  })

  test('identical patterns overlap', () => {
    expect(ipPatternsOverlap('192.168.1.*', '192.168.1.*')).toBe(true)
  })

  test('global wildcard overlaps any ip', () => {
    expect(ipPatternsOverlap('*.*.*.*', '1.2.3.4')).toBe(true)
  })
})
