import Vue from 'vue';
import App from './App.vue';
import router from './router';
import Buefy from 'buefy';
import '@creativebulma/bulma-tooltip/dist/bulma-tooltip.min.css';
import {FontAwesomeIcon} from '@fortawesome/vue-fontawesome';
import Course from './types/Course';
import Queue from './types/Queue';

Vue.component('font-awesome-icon', FontAwesomeIcon);

Vue.use(Buefy, {defaultIconPack: 'fa'});

Vue.config.productionTip = false;

export default new Vue({
	router,
	render: (h) => h(App),
	data: {
		courses: {} as {[index: string]: Course},
		queues: {} as {[index: string]: Queue},
		userInfoLoaded: false,
		loggedIn: false,
		userInfo: {},
	}
}).$mount('#app');
