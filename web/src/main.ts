import { createApp } from "vue";
import ElementPlus from "element-plus";
import zhCn from "element-plus/es/locale/lang/zh-cn";
import "element-plus/dist/index.css";
import "element-plus/theme-chalk/dark/css-vars.css";
import "@fontsource-variable/manrope";
import App from "./App.vue";
import router from "./router";
import { toast } from "./stores/ui";
import "./assets/main.css";

const app = createApp(App).use(router).use(ElementPlus, { locale: zhCn });
app.config.errorHandler = (error) =>
  toast((error as Error).message || "操作失败", { tone: "danger" });
app.mount("#app");
