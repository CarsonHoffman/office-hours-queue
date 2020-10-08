import Vue from 'vue';
import VueRouter, {RouteConfig} from 'vue-router';
import Home from '../views/Home.vue';
import Queue from '../views/Queue.vue';

Vue.use(VueRouter);

const routes: Array<RouteConfig> = [
	{
		path: '/',
		name: 'Home',
		component: Home,
		meta: {
			title: 'EECS Office Hours',
		}
	},
	{
		path: '/courses/:cid/queues/:qid',
		name: 'Queue',
		component: Queue,
	},
];

const router = new VueRouter({
	mode: 'history',
	base: process.env.BASE_URL,
	routes,
});

router.beforeEach((to, _, next) => {
	if (to.meta !== undefined && to.meta.title !== undefined) {document.title = to.meta.title};

	next();
})

export default router;
