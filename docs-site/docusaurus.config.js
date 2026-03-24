const config = {
  title: 'Araneae Docs',
  tagline: 'Araneae documentation site',
  url: 'https://example.com',
  baseUrl: '/Araneae/',
  onBrokenLinks: 'warn',
  onBrokenMarkdownLinks: 'warn',
  favicon: 'data:;base64,iVBORw0KGgo=',
  organizationName: 'example',
  projectName: 'Araneae',
  i18n: {
    defaultLocale: 'zh-Hans',
    locales: ['zh-Hans', 'en']
  },
  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.js',
          routeBasePath: '/'
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css'
        }
      }
    ]
  ],
  themeConfig: {
    navbar: {
      title: 'Araneae Docs',
      items: [{ type: 'docSidebar', sidebarId: 'tutorialSidebar', position: 'left', label: 'Docs' }]
    },
    footer: {
      style: 'dark',
      copyright: `Copyright © ${new Date().getFullYear()} Araneae`
    }
  }
};

module.exports = config;
