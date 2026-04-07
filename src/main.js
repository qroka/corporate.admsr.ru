import { createApp } from 'vue';
import ui from '@nuxt/ui/vue-plugin';
import { createPinia } from 'pinia';
import App from './App.vue';
import './assets/tailwind.css';
import { router } from './router';
import { applyUiTheme, getSavedUiTheme } from './composables/useUiTheme';

const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(ui);

applyUiTheme(getSavedUiTheme());

app.mount('#app');
