describe('Auth routes', () => {
  it('renders login page and switches to register and forgot views', () => {
    cy.visit('/login');
    cy.contains('Welcome Back');

    // Switch to register
    cy.contains("Sign up").click();
    cy.contains('Create Account');

    // Switch to forgot
    cy.contains('Sign in').click();
    cy.contains('Forgot your password?').click();
    cy.contains('Reset Password');
  });
});
