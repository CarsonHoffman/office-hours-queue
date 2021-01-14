<template>
	<li v-if="course.queues.length != 1">
		<a
			:class="{
				'is-active': courseHighlight,
				disabled: course.queues.length != 1,
			}"
			>{{ course.shortName }}</a
		>
		<ul>
			<router-link
				v-for="queue in course.queues"
				:key="queue.id"
				:to="'/queues/' + queue.id"
				:class="{ 'is-active': $route.path.includes(queue.id) }"
				:title="$route.path"
				>{{ queue.name }}</router-link
			>
		</ul>
	</li>
	<router-link
		v-else
		:to="'/queues/' + course.queues[0].id"
		:class="{
			'is-active': courseHighlight,
			disabled: course.queues.length == 0,
		}"
		>{{ course.shortName }}</router-link
	>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import Course from '../types/Course';
import Queue from '@/types/Queue';

@Component
export default class CourseNavbarItem extends Vue {
	@Prop({ required: true }) readonly course!: Course;

	get courseHighlight(): boolean {
		return this.course.queues.some((q: Queue) =>
			this.$route.path.includes(q.id)
		);
	}
}
</script>

<style scoped>
.disabled {
	color: currentColor;
	cursor: s-resize;
}
</style>
