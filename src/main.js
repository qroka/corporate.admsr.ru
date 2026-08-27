import { createApp } from 'vue';
import ui from '@nuxt/ui/vue-plugin';
import { createPinia } from 'pinia';
import App from './App.vue';
import './assets/tailwind.css';
import { router } from './router';
import { applyUiTheme, getSavedUiTheme } from './composables/useUiTheme';
import { applyKioskColorModeFromStorage } from './composables/useColorModeSchedule';
import { applyMainColorModeFromStorage } from './composables/useColorMode';
import { ensureSessionToken } from './composables/useAuthSession';

if (window.location.pathname.startsWith('/kiosk')) {
  applyKioskColorModeFromStorage();
} else {
  applyMainColorModeFromStorage();
}

const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(ui);

applyUiTheme(getSavedUiTheme());

// Выдать sessionToken для модуля курсов, если вход был до деплоя V4
void ensureSessionToken();

app.mount('#app');
