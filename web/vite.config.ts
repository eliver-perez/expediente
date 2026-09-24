import { defineConfig } from 'vite';
import { writeFileSync } from 'node:fs';

export default defineConfig({
  build: { outDir: '../internal/httpapi/assets', emptyOutDir: true, sourcemap: false },
  plugins: [{ name: 'retain-embed-directory', closeBundle() {
    writeFileSync(new URL('../internal/httpapi/assets/.gitkeep', import.meta.url), '');
  } }],
  server: { proxy: { '/api': { target: 'http://127.0.0.1:8090', changeOrigin: true,
    configure(proxy) { proxy.on('proxyReq', (outgoing, incoming) => {
      if (incoming.headers.origin === 'http://127.0.0.1:5173') outgoing.setHeader('Origin', 'http://127.0.0.1:8090');
    }); }
  } } }
});
