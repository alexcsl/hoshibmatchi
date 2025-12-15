import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import router from "./router";
import "./styles/main.scss";
import logger from "./utils/logger";

// Log application startup
logger.info("Starting Hoshi Vue application...");
logger.debug("Environment:", import.meta.env.MODE);
logger.debug("API URL:", import.meta.env.VITE_API_URL);

const app = createApp(App);
const pinia = createPinia();

app.use(pinia);
app.use(router);

// Log before mounting
logger.debug("Mounting Vue application...");
app.mount("#app");
logger.info("Hoshi Vue application mounted successfully");
