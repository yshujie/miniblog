import { createApp } from 'vue';

import App from './App.vue';
import router from './router';
import { setupStore } from './store';

import 'element-plus/es/components/message/style/css';
import '@/styles/index.scss';
import SvgIcon from './icons'; // icon
import './permission'; // permission control
import { checkEnableLogs } from './utils/error-log'; // error log

const app = createApp(App);
setupStore(app);
app.use(router);
app.component('svg-icon', SvgIcon);
checkEnableLogs(app);

app.mount('#app');
