import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

export default {
  // .svelte ファイル内の lang="ts" を処理するために必要
  preprocess: vitePreprocess(),
};
