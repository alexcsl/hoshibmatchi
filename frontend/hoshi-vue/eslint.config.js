import js from "@eslint/js";
import pluginVue from "eslint-plugin-vue";
import globals from "globals";
import ts from "typescript-eslint";

export default ts.config(
  js.configs.recommended,
  ...ts.configs.recommended,
  ...pluginVue.configs["flat/recommended"],
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
  },
  {
    files: ["**/*.vue", "**/*.ts", "**/*.tsx"],
    languageOptions: {
      parserOptions: {
        parser: ts.parser,
        ecmaVersion: "latest",
        sourceType: "module",
      },
    },
  },
  {
    rules: {
      // Enforce consistent spacing inside mustache interpolations (e.g., {{ value }}).
      "vue/mustache-interpolation-spacing": ["error", "always"],

      // Disallow spaces around equal signs in HTML attributes (e.g., <div class="box"> instead of <div class = "box">).
      "vue/no-spaces-around-equal-signs-in-attribute": "error",

      // Enforce consistent use of double quotes in HTML attributes.
      "vue/html-quotes": ["error", "double"],

      // Enforce consistent spacing before closing brackets in HTML tags.
      "vue/html-closing-bracket-spacing": [
        "error",
        {
          // No space before closing bracket of start tag (e.g., <div> not <div >).
          startTag: "never",
          // No space before closing bracket of end tag (e.g., </div> not </div >).
          endTag: "never",
          // Always space before self-closing tags (e.g., <img />).
          selfClosingTag: "always",
        },
      ],

      // Enforce self-closing style for components without content.
      "vue/html-self-closing": [
        "error",
        {
          html: {
            void: "always",
            normal: "never",
            component: "always",
          },
          svg: "always",
          math: "always",
        },
      ],

      // Enforce consistent spacing in HTML comments.
      "vue/html-comment-content-spacing": ["error", "always"],

      // Prefer const declarations where variables are not reassigned.
      "prefer-const": [
        "error",
        {
          destructuring: "any",
        },
      ],

      // Enforce v-bind shorthand syntax (e.g., :value instead of v-bind:value).
      "vue/v-bind-style": ["error", "shorthand"],

      // Enforce v-on shorthand syntax (e.g., @click instead of v-on:click).
      "vue/v-on-style": ["error", "shorthand"],

      // Enforce v-slot shorthand syntax (e.g., #default instead of v-slot:default).
      "vue/v-slot-style": ["error", "shorthand"],

      // Enforce consistent attribute order in Vue components.
      "vue/attributes-order": [
        "error",
        {
          order: [
            "DEFINITION",
            "LIST_RENDERING",
            "CONDITIONALS",
            "RENDER_MODIFIERS",
            "GLOBAL",
            "UNIQUE",
            "SLOT",
            "TWO_WAY_BINDING",
            "OTHER_DIRECTIVES",
            "OTHER_ATTR",
            "EVENTS",
            "CONTENT",
          ],
        },
      ],

      // Enforce component name casing in template (PascalCase).
      "vue/component-name-in-template-casing": ["error", "PascalCase"],

      // Disallow usage of `v-html` to prevent XSS.
      "vue/no-v-html": "warn",

      // Require semicolons at the end of statements.
      semi: ["error", "always"],

      // Enforce double quotes for string literals in TypeScript and JavaScript code.
      quotes: ["error", "double"],

      // Disallow unused variables in TypeScript (relaxed to warning for development).
      "@typescript-eslint/no-unused-vars": ["warn"],

      // Not allowed to use any (relaxed to warning for development).
      "@typescript-eslint/no-explicit-any": ["warn"],

      // Warn when `console.log` or similar are used, but allow console.warn and console.error.
      "no-console": ["warn", { allow: ["warn", "error"] }],

      // Allow single-word component names for pages (Admin, Login, Feed, etc.)
      "vue/multi-word-component-names": ["error", {
        ignores: ["Admin", "Archive", "Explore", "Feed", "Login", "Messages", "Profile", "Reels", "Settings", "Sidebar"]
      }],
    },
  }
);