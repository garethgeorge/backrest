import { paraglideVitePlugin } from '@inlang/paraglide-js'
import { defineConfig, loadEnv, type PluginOption } from 'vite';
import react from '@vitejs/plugin-react';
import viteCompression from 'vite-plugin-compression';
import { visualizer } from 'rollup-plugin-visualizer';

function renderChunks(id: string) {
  if (id.includes('node_modules')) {
    if (id.includes('node_modules/react/') ||
      id.includes('node_modules/react-dom/') ||
      id.includes('node_modules/react-router/') ||
      id.includes('node_modules/scheduler/')) {
      return 'react-vendor';
    }
    if (id.includes('node_modules/antd/') ||
      id.includes('node_modules/@ant-design/') ||
      id.includes('node_modules/rc-')) {
      return 'antd';
    }
    if (id.includes('node_modules/@connectrpc/') ||
      id.includes('node_modules/@bufbuild/')) {
      return 'connectrpc';
    }
    if (id.includes('node_modules/recharts/') ||
      id.includes('node_modules/d3-')) {
      return 'recharts';
    }
    return 'vendor';
  }
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');

  return {
    plugins: [paraglideVitePlugin({
      project: './project.inlang', outdir: './src/paraglide', strategy: [
        'preferredLanguage',
        'baseLocale',
      ]
    }),
    react(),
    viteCompression({ algorithm: 'gzip', ext: '.gz', deleteOriginFile: true }),
    // viteCompression({ algorithm: 'brotliCompress', ext: '.br' }),
    visualizer({
      open: false,
      gzipSize: true,
      // brotliSize: true,
    }) as PluginOption,
    ],
    base: './',
    build: {
      outDir: 'dist',
      target: 'esnext',
      minify: 'esbuild',
      rollupOptions: {
        output: {
          manualChunks: renderChunks
        },
      },
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