import { defineConfig } from 'vite'

export default defineConfig({
  server: {
    proxy: {
      // forwards http://localhost:5173/api/* -> http://localhost:8080/api/*
      '/api': 'http://localhost:8080',
      // forwards seamark tiles too so your style can stay relative
      '/seamark': 'http://localhost:8080',
    },
  },
})
