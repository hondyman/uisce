declare module '@testing-library/react' {
  // Minimal shim for testing-library exports used in the test suite
  export const render: any;
  export const screen: any;
  export const fireEvent: any;
  export const waitFor: any;
  export const within: any;
  export const cleanup: any;
  const rtl: any;
  export default rtl;
}
