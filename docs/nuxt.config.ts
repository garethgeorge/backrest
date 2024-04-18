export default defineNuxtConfig({
  extends: ['@nuxt-themes/docus'],
  devtools: { enabled: true },
  ssr: true,
  app: {
    baseURL: '/backrest/',
  }
})
