import Vue from 'vue';
import VueRouter, { RouteConfig } from 'vue-router';
import Home from '../views/Home.vue';
import Queue from '../views/Queue.vue';
import AdminPage from '@/views/Admin.vue';

Vue.use(VueRouter);

const routes: Array<RouteConfig> = [
	{
		path: '/',
		name: 'Home',
		component: Home,
		meta: {
			title: 'EECS Office Hours',
		},
	},
	{
		path: '/queues/:qid',
		name: 'Queue',
		component: Queue,
	},
	{
		path: '/admin',
		name: 'Admin',
		component: AdminPage,
		meta: {
			title: 'Course Admin Controls',
		},
	},
];

const router = new VueRouter({
	mode: 'history',
	base: process.env.BASE_URL,
	routes,
});

router.beforeEach((to, _, next) => {
	if (to.meta !== undefined && to.meta.title !== undefined) {
		document.title = to.meta.title;
	}

	next();
});

export default router;
