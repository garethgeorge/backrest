import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { defineConfig, loadEnv, type PluginOption } from 'vite';
import react from '@vitejs/plugin-react';
import tsconfigPaths from 'vite-tsconfig-paths';
import viteCompression from 'vite-plugin-compression';
// import { visualizer } from 'rollup-plugin-visualizer';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');

  return {
    plugins: [
      paraglideVitePlugin({
        project: './project.inlang',
        outdir: './src/paraglide',
        strategy: ['localStorage', 'preferredLanguage', 'baseLocale'],
      }),
      react(),
      tsconfigPaths(),
      viteCompression({ algorithm: 'gzip', ext: '.gz', deleteOriginFile: true }),
      // viteCompression({ algorithm: 'brotliCompress', ext: '.br' }),
      // visualizer({
      //   open: false,
      //   gzipSize: true,
      //   // brotliSize: true,
      // }) as PluginOption,
    ],
    base: './',
    build: {
      outDir: 'dist',
      target: 'esnext',
      minify: 'esbuild',
    },
    define: {
      'process.env.UI_OS': JSON.stringify(env.UI_OS),
      'process.env.BACKREST_BUILD_VERSION': JSON.stringify(env.BACKREST_BUILD_VERSION),
      'process.env.UI_BACKEND_URL': JSON.stringify(env.UI_BACKEND_URL),
      'process.env.UI_FEATURES': JSON.stringify(env.UI_FEATURES),
    },
    server: {
      proxy: {
        '/v1.Backrest': {
          target: env.UI_BACKEND_URL || 'http://localhost:9898',
          secure: false,
        },
        '/v1.Authentication': {
          target: env.UI_BACKEND_URL || 'http://localhost:9898',
          secure: false,
        },
        '/download': {
          target: env.UI_BACKEND_URL || 'http://localhost:9898',
          secure: false,
        },
      },
    },
  };
});
