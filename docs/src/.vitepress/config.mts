import { defineConfig } from 'vitepress'

export default defineConfig({
  title: "Backrest",
  description: "Web UI and orchestrator for restic backup",
  base: "/backrest/",
  cleanUrls: true,
  themeConfig: {
    logo: '/logo.svg', // Assuming there's a logo or comment out if none
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Docs', link: '/introduction/getting-started' },
    ],

    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: 'Getting Started', link: '/introduction/getting-started' },
          { text: 'Restore Files', link: '/introduction/restore-files' }
        ]
      },
      {
        text: 'Documentation',
        items: [
          { text: 'Operations', link: '/docs/operations' },
          { text: 'Hooks', link: '/docs/hooks' },
          { text: 'API', link: '/docs/api' }
        ]
      },
      {
        text: 'Cookbooks',
        items: [
          { text: 'Command Hook Examples', link: '/cookbooks/command-hook-examples' },
          { text: 'Reverse Proxy Examples', link: '/cookbooks/reverse-proxy-examples' },
          { text: 'Slack Hook (Build Kit)', link: '/cookbooks/slack-hook-build-kit-examples' },
          { text: 'SSH Remote', link: '/cookbooks/ssh-remote' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/garethgeorge/backrest' }
    ],

    editLink: {
      pattern: 'https://github.com/garethgeorge/backrest/edit/main/docs/src/:path',
      text: 'Edit this page on GitHub'
    }
  }
})
