/**
 * @file main.ts
 * @description Main entrypoint of the frontend application.
 * Initializes the Vue App instance, configures the global state manager (Pinia),
 * imports global styles, and mounts the application on the HTML element `#app`.
 */

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './style.css'

// every theme file of ./themes: adding a theme is adding a file there (its roles win over the contract's by
// specificity, so the order they load in does not matter)
import.meta.glob('./themes/*.css', { eager: true })
import App from './App.vue'

const app = createApp(App)
app.use(createPinia())
app.mount('#app')
