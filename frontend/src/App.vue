<template>
	<div>
		<nav class="navbar has-shadow is-spaced">
			<div class="container">
				<div class="navbar-brand">
					<div class="navbar-item">
						<h1 class="title">
							<router-link to="/">EECS Office Hours</router-link>
						</h1>
					</div>
				</div>
				<div class="navbar-end">
					<div class="navbar-item">
						<button class="button is-info is-loading" v-if="!$root.$data.userInfoLoaded">Log in</button>
						<button class="button is-info" v-else-if="!$root.$data.loggedIn">Log in</button>
						<button class="button is-danger" v-else>Log out</button>
					</div>
				</div>
			</div>
		</nav>
		<section class="section main-section">
			<div class="container">
				<div class="columns" v-if="fetchedCourses">
					<div class="column is-one-fifth">
						<b-menu>
							<b-menu-list label="Courses">
								<course-navbar-item v-for="course in courses" :course="course" :key="course.id" />
							</b-menu-list>
						</b-menu>
					</div>
					<div class="column is-four-fifths">
						<transition name="fade" mode="out-in">
							<router-view :key="$route.fullPath"></router-view>
						</transition>
					</div>
				</div>
				<b-loading active="true" v-else></b-loading>
			</div>
		</section>
	</div>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';

import CourseNavbarItem from '@/components/CourseNavbarItem.vue';
import Course from './types/Course';

@Component({ components: { CourseNavbarItem } })
export default class App extends Vue {
	@Prop() fetchedCourses = false;

	created() {
		fetch(process.env.BASE_URL + 'api/courses')
			.then((resp) => resp.json())
			.then((data) => {
				data.map((c: any) => {
					const course = new Course(c);
					Vue.set(this.$root.$data.courses, c.id, course);
					for (const q of course.queues) {
						Vue.set(this.$root.$data.queues, q.id, q);
					}
					this.fetchedCourses = true;
				});
			});

		fetch(process.env.BASE_URL + 'api/users/@me')
			.then((resp) => {
				if (resp.status !== 200) {
					return Promise.reject('not logged in');
				}
				return resp.json();
			})
			.then((data) => {
				Vue.set(this.$root.$data, 'userInfoLoaded', true);
				Vue.set(this.$root.$data, 'loggedIn', true);
				Vue.set(this.$root.$data, 'userInfo', data);
			})
			.catch((p) => (this.$root.$data.userInfoLoaded = true));
	}

	get courses() {
		return Object.values(this.$root.$data.courses).filter(
			(c: Course) => c.queues.length > 0
		);
	}
}
</script>

<style lang="scss">
@import '~bulma/sass/utilities/_all';

$primary: #167df0;
$menu-item-active-background-color: $primary;

// Setup $colors to use as bulma classes
$colors: (
	'white': (
		$white,
		$black,
	),
	'black': (
		$black,
		$white,
	),
	'light': (
		$light,
		$light-invert,
	),
	'dark': (
		$dark,
		$dark-invert,
	),
	'primary': (
		$primary,
		$primary-invert,
	),
	'info': (
		$info,
		$info-invert,
	),
	'success': (
		$success,
		$success-invert,
	),
	'warning': (
		$warning,
		$warning-invert,
	),
	'danger': (
		$danger,
		$danger-invert,
	),
);

@import '~bulma';
@import '~buefy/src/scss/buefy';

.router-link-active {
	text-decoration: none;
	color: inherit;
}

.fade-enter-active,
.fade-leave-active {
	transition-duration: 0.2s;
	transition-property: opacity;
	transition-timing-function: ease;
}

.fade-enter,
.fade-leave-active {
	opacity: 0;
}
</style>
