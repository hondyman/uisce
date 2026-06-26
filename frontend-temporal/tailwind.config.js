/**
 * Tailwind config for frontend-temporal — adds wf-pulse keyframes, animation and boxShadow for node highlight
 */
module.exports = {
  content: [
    './index.html',
    './src/**/*.{js,ts,jsx,tsx,html}'
  ],
  theme: {
    extend: {
      keyframes: {
        'wf-pulse': {
          '0%': { transform: 'scale(1)' },
          '50%': { transform: 'scale(1.02)' },
          '100%': { transform: 'scale(1)' }
        }
      },
      // Use CSS variable for duration/iterations so we can toggle via utilities
      animation: {
        'wf-pulse': 'wf-pulse var(--wf-pulse-duration, 900ms) ease-in-out 0s var(--wf-pulse-iterations, 3)'
      },
      boxShadow: {
        'wf-highlight': '0 0 0 6px rgba(67, 75, 234, 0.14)'
      }
    }
  },
  plugins: [
    function({ addUtilities }) {
      const newUtilities = {
        '.wf-duration-300': { '--wf-pulse-duration': '300ms' },
        '.wf-duration-600': { '--wf-pulse-duration': '600ms' },
        '.wf-duration-900': { '--wf-pulse-duration': '900ms' },
        '.wf-iterations-1': { '--wf-pulse-iterations': '1' },
        '.wf-iterations-3': { '--wf-pulse-iterations': '3' }
      }
      addUtilities(newUtilities, { variants: ['responsive'] })
    }
  ]
}
