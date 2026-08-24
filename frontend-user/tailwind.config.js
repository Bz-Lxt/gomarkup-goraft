/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        void: '#07090d',
        panel: '#10151d',
        grid: '#1b3a3a',
        leader: '#e8b84a',
        follower: '#3ee0c6',
        candidate: '#ff8a3d',
        down: '#6b7280',
        danger: '#ff4d6d',
        mute: '#7f8b99',
      },
      fontFamily: {
        display: ['"Chakra Petch"', 'sans-serif'],
        sans: ['"IBM Plex Sans"', 'sans-serif'],
        mono: ['"IBM Plex Mono"', 'monospace'],
      },
    },
  },
  plugins: [],
}
