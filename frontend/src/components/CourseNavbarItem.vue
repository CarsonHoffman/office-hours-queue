<template>
	<li v-if="course.queues.length != 1">
		<a
			:class="{ 'is-active': $route.path.includes(course.id), disabled: course.queues.length != 1 }"
		>{{ course.shortName }}</a>
		<ul>
			<router-link
				v-for="queue in course.queues"
				:key="queue.id"
				:to="'/courses/'+course.id+'/queues/'+queue.id"
				:class="{ 'is-active': $route.path.includes(queue.id) }"
				:title="$route.path"
			>{{ queue.name }}</router-link>
		</ul>
	</li>
	<router-link
		v-else
		:to="'/courses/'+course.id+'/queues/'+course.queues[0].id"
		:class="{ 'is-active': $route.path.includes(course.id), disabled: course.queues.length == 0 }"
	>{{ course.shortName }}</router-link>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import Course from '../types/Course';

@Component
export default class CourseNavbarItem extends Vue {
	@Prop({ required: true }) readonly course!: Course;
}
</script>

<style scoped>
.disabled {
	color: currentColor;
	cursor: s-resize;
}
</style>
