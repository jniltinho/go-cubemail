/**
 * @file main.ts
 * @description Main entrypoint of the frontend application.
 * Initializes the Vue App instance, configures the global state manager (Pinia),
 * imports global styles, and mounts the application on the HTML element `#app`.
 */

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './style.css'
import App from './App.vue'

const app = createApp(App)
app.use(createPinia())
app.mount('#app')
