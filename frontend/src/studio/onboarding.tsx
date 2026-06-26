
export function Onboarding({ kernel }) {
  const start = () => {
    kernel.services.persistence.markOnboardingComplete()
    kernel.events.dispatch("onboarding.complete")
  }

  return (
    <div className="onboarding">
      <h1>Welcome to Rule Studio</h1>
      <p>This environment lets you write, simulate, debug, and promote ASL rules.</p>
      <button onClick={start}>Get Started</button>
    </div>
  )
}