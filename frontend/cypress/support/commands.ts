/// <reference types="cypress" />
// custom commands can be added here
Cypress.Commands.add('loginIfNeeded', () => {
  // implement if your app requires authentication
});

// Declare the custom command on Cypress. This lets TypeScript accept
// `Cypress.Commands.add('loginIfNeeded', ...)` without complaining.
declare global {
  namespace Cypress {
    interface Chainable {
      /**
       * Custom command which logs in if needed.
       */
      loginIfNeeded(): Chainable<void>;
    }
  }
}

export {};
