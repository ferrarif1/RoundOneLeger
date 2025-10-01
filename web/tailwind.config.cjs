const defaultTheme = require('tailwindcss/defaultTheme');

module.exports = {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['"Plus Jakarta Sans"', ...defaultTheme.fontFamily.sans]
      },
      colors: {
        night: {
          50: '#1a1d2d',
          100: '#2c3147',
          200: '#454b68',
          300: '#6a7191',
          400: '#8f95b2',
          500: '#b5b9cf',
          600: '#d4d6e5',
          700: '#e6e8f2',
          800: '#f1f2f8',
          900: '#f8f8fb'
        },
        neon: {
          400: '#fbc28d',
          500: '#f89b54',
          600: '#ef8140'
        },
        ink: {
          50: '#f5f6fb',
          200: '#e2e5f1',
          400: '#c5cadb',
          600: '#98a0ba',
          800: '#4a4f63',
          900: '#12131c'
        }
      },
      boxShadow: {
        glow: '0 12px 30px rgba(26, 29, 45, 0.08)'
      },
      backgroundImage: {
        'eidos-radial': 'radial-gradient(circle at 50% -20%, rgba(248, 155, 84, 0.18), transparent 65%)'
      }
    }
  },
  plugins: [require('@tailwindcss/forms')]
};
