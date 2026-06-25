/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./web/templates/**/*.html'],
  theme: {
    extend: {
      colors: {
        accent: {
          DEFAULT: '#0066cc',
          light: '#4d9fff',
          subtle: 'rgba(0, 102, 204, 0.08)',
        },
        surface: {
          DEFAULT: '#ffffff',
          hover: '#f5f5f5',
          code: '#f6f8fa',
          inline: '#f3f4f5',
        },
        muted: {
          DEFAULT: '#666666',
          light: '#aaaaaa',
          border: '#eaeaea',
        },
      },
      fontFamily: {
        mono: ['SF Mono', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'Liberation Mono', 'monospace'],
      },
      boxShadow: {
        nav: '0 2px 10px rgba(0, 0, 0, 0.05)',
        card: '0 2px 8px rgba(0, 0, 0, 0.04)',
        modal: '0 10px 25px rgba(0, 0, 0, 0.2)',
      },
      borderRadius: {
        card: '12px',
      },
    },
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}
