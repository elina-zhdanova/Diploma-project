// Должен совпадать с HTTP_PORT API (по умолчанию 8080, см. backend internal/config).
// Если API в Docker с пробросом хоста 8081:8080 — задайте ITSHOP_API_PROXY=http://localhost:8081
// В Docker-сервисе web: ITSHOP_API_PROXY=http://api:8080 (уже в compose).
const target = process.env.ITSHOP_API_PROXY || 'http://localhost:8080';

module.exports = {
  '/api': {
    target,
    secure: false,
    changeOrigin: true,
  },
  '/health': {
    target,
    secure: false,
    changeOrigin: true,
  },
  '/internal': {
    target,
    secure: false,
    changeOrigin: true,
  },
};
