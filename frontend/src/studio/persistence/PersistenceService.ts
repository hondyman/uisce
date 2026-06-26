export class PersistenceService {
  save(kernel) {
    localStorage.setItem("asl.rule", kernel.state.rule)
    localStorage.setItem("asl.context", JSON.stringify(kernel.state.context))
  }

  restore(kernel) {
    kernel.state.rule = localStorage.getItem("asl.rule") || ""
    kernel.state.context = JSON.parse(localStorage.getItem("asl.context") || "{}")
  }

  hasSeenOnboarding() {
    return localStorage.getItem("asl.onboarding.complete") === "true"
  }

  markOnboardingComplete() {
    localStorage.setItem("asl.onboarding.complete", "true")
  }

  saveVersion(content, type = "manual") {
    const versions = this.getVersions()
    const version = {
      id: Date.now().toString(),
      timestamp: Date.now(),
      content,
      type
    }
    versions.unshift(version)
    // Keep only last 50 versions
    if (versions.length > 50) {
      versions.splice(50)
    }
    localStorage.setItem("asl.versions", JSON.stringify(versions))
  }

  getVersions() {
    try {
      return JSON.parse(localStorage.getItem("asl.versions") || "[]")
    } catch {
      return []
    }
  }
}