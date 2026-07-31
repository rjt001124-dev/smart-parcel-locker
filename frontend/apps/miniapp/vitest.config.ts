import { defineConfig } from "vitest/config";

export default defineConfig({
  define: {
    ENABLE_ADJACENT_HTML: false,
    ENABLE_CLONE_NODE: false,
    ENABLE_CONTAINS: false,
    ENABLE_INNER_HTML: true,
    ENABLE_MUTATION_OBSERVER: false,
    ENABLE_SIZE_APIS: false,
    ENABLE_TEMPLATE_CONTENT: false
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"]
  }
});
