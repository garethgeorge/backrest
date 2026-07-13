import { defineConfig } from 'vitepress'

export default defineConfig({
  title: "Backrest",
  description: "Web UI and orchestrator for restic backup",
  base: "/backrest/",
  cleanUrls: true,
  lastUpdated: true,
  sitemap: {
    hostname: 'https://garethgeorge.github.io/backrest/'
  },
  themeConfig: {
    logo: '/logo.svg',
    search: {
      provider: 'local'
    },
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Guide', link: '/introduction/getting-started' },
      { text: 'Reference', link: '/docs/operations' },
    ],

    sidebar: [
      {
        text: 'Getting Started',
        items: [
          { text: 'Introduction & Concepts', link: '/introduction/getting-started' },
          { text: 'Installation', link: '/introduction/installation' },
          { text: 'Your First Backup', link: '/introduction/first-backup' },
          { text: 'Restoring Files', link: '/introduction/restore-files' }
        ]
      },
      {
        text: 'Guides',
        items: [
          { text: 'Scheduling Backups', link: '/guides/scheduling' },
          { text: 'Retention & Repo Health', link: '/guides/repo-health' },
          { text: 'Storage Backends', link: '/guides/storage-backends' },
          { text: 'SFTP & SSH Remotes', link: '/guides/sftp' },
          { text: 'Notifications', link: '/guides/notifications' },
          { text: 'Authentication & Security', link: '/guides/security' }
        ]
      },
      {
        text: 'Reference',
        items: [
          { text: 'Operational Model', link: '/docs/operations' },
          { text: 'Configuration & Paths', link: '/docs/configuration' },
          { text: 'Hooks', link: '/docs/hooks' },
          { text: 'Multihost Sync', link: '/docs/multihost' },
          { text: 'API', link: '/docs/api' }
        ]
      },
      {
        text: 'Cookbooks',
        items: [
          { text: 'Command Hook Examples', link: '/cookbooks/command-hook-examples' },
          { text: 'Reverse Proxies', link: '/cookbooks/reverse-proxy-examples' },
          { text: 'Slack Hook (Block Kit)', link: '/cookbooks/slack-hook-build-kit-examples' },
          { text: 'SFTP in Docker (Manual Keys)', link: '/cookbooks/ssh-remote' }
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
