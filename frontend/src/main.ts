import Vue from 'vue';
import App from './App.vue';
import router from './router';
import Buefy from 'buefy';
import '@creativebulma/bulma-tooltip/dist/bulma-tooltip.min.css';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import Course from './types/Course';
import Queue from './types/Queue';
import moment from 'moment-timezone';

Vue.component('font-awesome-icon', FontAwesomeIcon);

import { library } from '@fortawesome/fontawesome-svg-core';
import {
	faCheck,
	faCheckCircle,
	faInfoCircle,
	faExclamationTriangle,
	faExclamationCircle,
	faArrowUp,
	faAngleRight,
	faAngleLeft,
	faAngleDown,
	faEye,
	faEyeSlash,
	faCaretDown,
	faCaretUp,
	faUpload,
	faEnvelopeOpenText,
	faBullhorn,
} from '@fortawesome/free-solid-svg-icons';

library.add(
	faCheck,
	faCheckCircle,
	faInfoCircle,
	faExclamationTriangle,
	faExclamationCircle,
	faArrowUp,
	faAngleRight,
	faAngleLeft,
	faAngleDown,
	faEye,
	faEyeSlash,
	faCaretDown,
	faCaretUp,
	faUpload,
	faEnvelopeOpenText,
	faBullhorn
);

Vue.use(Buefy, {
	defaultIconComponent: 'font-awesome-icon',
	defaultIconPack: 'fas',
});

Vue.config.productionTip = false;

moment.relativeTimeThreshold('m', 60);
moment.relativeTimeThreshold('d', 24);

export default new Vue({
	router,
	render: (h) => h(App),
	data: {
		courses: {} as { [index: string]: Course },
		showCourses: false,
		queues: {} as { [index: string]: Queue },
		userInfoLoaded: false,
		loggedIn: false,
		userInfo: {},
		studentView: false,
	},
}).$mount('#app');
