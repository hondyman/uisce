// React import removed — hooks/JSX handled by the compiler/runtime
import BundleRecommendationPanel from '../../../components/BundleRecommendationPanel'
import './BundlesPage.css'

export default function BundlesPage(): JSX.Element {
  return (
    <div className="bundles-page">
      <h2>AI Bundle Recommendations</h2>
      <p className="lead">Suggested claim bundles generated from recent usage patterns. Review and approve to publish to the bundle catalog.</p>
      <BundleRecommendationPanel />
    </div>
  )
}
