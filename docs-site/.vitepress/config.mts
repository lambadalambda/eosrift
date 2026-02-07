import { defineConfig } from "vitepress";

export default defineConfig({
  title: "Eosrift Documentation",
  description: "Self-hosted ngrok-like tunneling docs",
  base: "/docs/",
  outDir: "../internal/server/docs_static",
  cleanUrls: true,
  themeConfig: {
    nav: [
      { text: "Getting Started", link: "/getting-started" },
      { text: "Client CLI", link: "/client-cli" },
      { text: "Server Admin", link: "/server-admin" }
    ],
    sidebar: [
      {
        text: "Guide",
        items: [
          { text: "Overview", link: "/" },
          { text: "Getting Started", link: "/getting-started" },
          { text: "Client CLI", link: "/client-cli" },
          { text: "Server Admin", link: "/server-admin" }
        ]
      }
    ],
    socialLinks: [{ icon: "github", link: "https://github.com/lambadalambda/eosrift" }]
  }
});
