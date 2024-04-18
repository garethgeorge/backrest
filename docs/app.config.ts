// https://github.com/nuxt-themes/docus/blob/main/nuxt.schema.ts
export default defineAppConfig({
  docus: {
    title: 'Backrest',
    description: 'Backrest is a web UI and orchestrator for restic backup.',
    // image: 'https://user-images.githubusercontent.com/904724/185365452-87b7ca7b-6030-4813-a2db-5e65c785bf88.png',
    socials: {
      github: 'garethgeorge/backrest',
    },
    github: {
      dir: 'docs/content',
      branch: 'main',
      repo: 'backrest',
      owner: 'garethgeorge',
      edit: true
    },
    aside: {
      level: 0,
      collapsed: false,
      exclude: []
    },
    main: {
      padded: true,
      fluid: true
    },
    header: {
      logo: false,
      showLinkIcon: true,
      exclude: [],
      fluid: true,
    }
  }
})
