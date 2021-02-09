<template>
	<div>
		<nav class="navbar has-shadow is-spaced">
			<div class="container">
				<div class="navbar-brand">
					<div class="navbar-item">
						<h1 class="title">
							<router-link to="/" class="no-link-color"
								>EECS Office Hours</router-link
							>
						</h1>
					</div>
				</div>
				<div class="navbar-end">
					<div class="navbar-item" v-if="admin">
						<b-tooltip class="is-left" label="Student View">
							<font-awesome-icon
								class="clickable-icon"
								icon="user-graduate"
								size="2x"
								@click="setStudentView(true)"
						/></b-tooltip>
					</div>
					<div class="navbar-item" v-if="studentView">
						<b-tooltip class="is-left" label="Exit Student View">
							<font-awesome-icon
								class="clickable-icon"
								icon="user-shield"
								size="2x"
								@click="setStudentView(false)"
							/>
						</b-tooltip>
					</div>
					<div class="navbar-item" v-if="admin">
						<router-link to="/admin" class="no-link-color">
							<font-awesome-icon icon="user-shield" size="2x" />
						</router-link>
					</div>
					<div class="navbar-item">
						<a
							href="https://github.com/CarsonHoffman/office-hours-queue"
							target="_blank"
							class="no-link-color"
						>
							<font-awesome-icon :icon="['fab', 'github']" size="2x" />
						</a>
					</div>
					<div class="navbar-item">
						<button
							class="button is-info is-loading"
							v-if="!$root.$data.userInfoLoaded"
						>
							<span class="icon"><font-awesome-icon icon="sign-in-alt"/></span>
							<span>Log in</span>
						</button>
						<a :href="loginUrl" v-else-if="!$root.$data.loggedIn">
							<button class="button is-info">
								<span class="icon"
									><font-awesome-icon icon="sign-in-alt"
								/></span>
								<span>Log in</span>
							</button>
						</a>
						<a :href="logoutUrl" v-else>
							<button class="button is-danger">
								<span class="icon"
									><font-awesome-icon icon="sign-out-alt"
								/></span>
								<span>Log out</span>
							</button>
						</a>
					</div>
				</div>
			</div>
		</nav>
		<section class="section main-section">
			<div class="container">
				<div class="columns" v-if="fetchedCourses">
					<div class="column is-one-fifth">
						<b-menu class="sticky" :activable="false">
							<b-menu-list label="Courses">
								<div class="course" v-for="course in courses" :key="course.id">
									<b-menu-item
										class="course-item"
										:label="course.shortName"
										:active="
											course.queues.some((q) => $route.path.includes(q.id))
										"
										:expanded="
											course.queues.some((q) => $route.path.includes(q.id))
										"
										v-if="course.queues.length > 1"
									>
										<b-menu-item
											class="course-item"
											v-for="queue in course.queues"
											:key="queue.id"
											tag="router-link"
											:to="'/queues/' + queue.id"
											:label="queue.name"
											:active="$route.path.includes(queue.id)"
										></b-menu-item
									></b-menu-item>
									<b-menu-item
										class="course-item"
										:label="course.shortName"
										:active="
											course.queues.some((q) => $route.path.includes(q.id))
										"
										tag="router-link"
										:to="'/queues/' + course.queues[0].id"
										v-else
									></b-menu-item
									><font-awesome-icon
										:icon="[course.favorite ? 'fas' : 'far', 'star']"
										class="clickable-icon course-favorite"
										:class="{
											'white-icon': course.queues.some((q) =>
												$route.path.includes(q.id)
											),
										}"
										@click="toggleFavorite(course)"
									/>
								</div>
							</b-menu-list>
						</b-menu>
					</div>
					<div class="column is-four-fifths">
						<transition name="fade" mode="out-in">
							<router-view
								:key="$route.fullPath"
								:studentView="studentView"
								@disconnected="restart"
							></router-view>
						</transition>
					</div>
				</div>
				<b-loading :active="true" v-else></b-loading>
			</div>
		</section>
	</div>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import Course from './types/Course';

import { library } from '@fortawesome/fontawesome-svg-core';
import {
	faSignInAlt,
	faSignOutAlt,
	faUserShield,
	faStar as solidStar,
	faUserGraduate,
} from '@fortawesome/free-solid-svg-icons';
import { faStar as regularStar } from '@fortawesome/free-regular-svg-icons';
import { faGithub } from '@fortawesome/free-brands-svg-icons';

library.add(
	faSignInAlt,
	faSignOutAlt,
	faUserShield,
	faGithub,
	solidStar,
	regularStar,
	faUserGraduate
);

@Component
export default class App extends Vue {
	@Prop() fetchedCourses = false;
	studentView = false;

	created() {
		if ('Notification' in window) {
			Notification.requestPermission();
		}

		this.restart();
	}

	get loginUrl() {
		return process.env.BASE_URL + 'api/oauth2login';
	}

	get logoutUrl() {
		return process.env.BASE_URL + 'api/logout';
	}

	get courses() {
		return Object.values(this.$root.$data.courses)
			.filter((c: Course) => c.queues.length > 0)
			.sort((a: Course, b: Course) => {
				if (a.favorite !== b.favorite) {
					return a.favorite ? -1 : 1;
				}
				return a.id < b.id ? -1 : 1;
			});
	}

	get admin() {
		return (
			!this.studentView &&
			this.$root.$data.loggedIn &&
			(this.$root.$data.userInfo.site_admin ||
				this.$root.$data.userInfo.admin_courses.length > 0)
		);
	}

	setStudentView(studentView: boolean) {
		this.studentView = studentView;
		this.$root.$data.studentView = studentView;
	}

	// Drop all courses and authorization information and re-start
	// the loading process. This is essentially a complete refresh
	// without actually refreshing the page.
	restart() {
		this.fetchedCourses = false;
		Vue.set(this.$root.$data, 'courses', {});
		Vue.set(this.$root.$data, 'queues', {});

		Vue.set(this.$root.$data, 'userInfoLoaded', false);
		Vue.set(this.$root.$data, 'loggedIn', false);
		Vue.set(this.$root.$data, 'userInfo', {});

		fetch(process.env.BASE_URL + 'api/courses')
			.then((resp) => resp.json())
			.then((data) => {
				data.map((c: any) => {
					const course = new Course(c);
					course.favorite =
						localStorage.getItem('favoriteCourses-' + course.id) !== null;
					Vue.set(this.$root.$data.courses, c.id, course);
					for (const q of course.queues) {
						Vue.set(this.$root.$data.queues, q.id, q);
					}
				});
				this.fetchedCourses = true;
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

	toggleFavorite(c: Course) {
		const original = c.favorite;
		if (original) {
			localStorage.removeItem('favoriteCourses-' + c.id);
		} else {
			localStorage.setItem('favoriteCourses-' + c.id, 'favorite');
		}
		c.favorite = !original;
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

.no-link-color {
	text-decoration: none;
	color: inherit;
}

.fade-enter-active,
.fade-leave-active {
	transition-duration: 0.1s;
	transition-property: opacity;
	transition-timing-function: ease;
}

.fade-enter,
.fade-leave-active {
	opacity: 0;
}

.sticky {
	/* Don't attempt to sticky on mobile */
	@media only screen and (min-width: 769px) {
		position: sticky;
		top: 1.5em;
	}
}

.course {
	display: flex;
	align-items: start;
}

.course-item {
	flex-grow: 1;
}

.clickable-icon {
	pointer-events: auto;
	cursor: pointer;
}

.course-favorite {
	position: absolute;
	right: 0;
	margin-right: 0.6em;
	margin-top: 0.6em;
}

.white-icon {
	color: white;
}
</style>
