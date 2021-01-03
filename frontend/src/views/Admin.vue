<template>
	<div>
		<p class="title">Courses</p>
		<button class="button is-primary is-fullwidth block" @click="addCourse">
			Add Course
		</button>
		<div class="box" v-for="(course, i) in courses" :key="i">
			<div class="level">
				<div class="level-left">
					<div class="level-item">
						<strong>{{ course.shortName }}</strong>
					</div>
					<div class="level-item">
						{{ course.fullName }}
					</div>
				</div>
				<div class="level-right">
					<div class="level-item">
						<b-tooltip label="Edit Course">
							<button class="button is-white" @click="editCourse(i)">
								<span class="icon"
									><font-awesome-icon icon="edit"
								/></span></button
						></b-tooltip>
					</div>
					<div class="level-item">
						<b-tooltip label="Add Queue">
							<button class="button is-white" @click="addQueue(i)">
								<span class="icon"
									><font-awesome-icon icon="plus"
								/></span></button
						></b-tooltip>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator';
import { library } from '@fortawesome/fontawesome-svg-core';
import { faEdit, faPlus } from '@fortawesome/free-solid-svg-icons';
import Course from '@/types/Course';
import CourseEdit from '@/components/admin/CourseEdit.vue';
import QueueAdd from '@/components/admin/QueueAdd.vue';
import ErrorDialog from '@/util/ErrorDialog';

library.add(faEdit, faPlus);

@Component
export default class AdminPage extends Vue {
	get courses(): Course[] {
		return Object.values(this.$root.$data.courses);
	}

	addCourse() {
		this.$buefy.modal.open({
			parent: this,
			component: CourseEdit,
			props: { defaultShortName: '', defaultFullName: '', defaultAdmins: [] },
			events: {
				saved: (short: string, full: string, admins: string[]) => {
					fetch(process.env.BASE_URL + `api/courses`, {
						method: 'POST',
						body: JSON.stringify({ short_name: short, full_name: full }),
					}).then((res) => {
						if (res.status !== 201) {
							return ErrorDialog(res);
						}

						if (admins.length > 0) {
							res.json().then((body: { [index: string]: unknown }) => {
								fetch(
									process.env.BASE_URL + `api/courses/${body['id']}/admins`,
									{ method: 'PUT', body: JSON.stringify(admins) }
								).then((res) => {
									if (res.status !== 204) {
										return ErrorDialog(res);
									}
									location.reload();
								});
							});
						} else {
							location.reload();
						}
					});
				},
			},
			hasModalCard: true,
			trapFocus: true,
		});
	}

	editCourse(index: number) {
		const course = this.courses[index];
		fetch(process.env.BASE_URL + `api/courses/${course.id}/admins`)
			.then((res) => res.json())
			.then((admins) => {
				this.$buefy.modal.open({
					parent: this,
					component: CourseEdit,
					props: {
						defaultShortName: this.courses[index].shortName,
						defaultFullName: this.courses[index].fullName,
						defaultAdmins: admins,
					},
					events: {
						saved: (short: string, full: string, admins: string[]) => {
							Promise.all([
								fetch(process.env.BASE_URL + `api/courses/${course.id}`, {
									method: 'PUT',
									body: JSON.stringify({ short_name: short, full_name: full }),
								}),
								fetch(
									process.env.BASE_URL + `api/courses/${course.id}/admins`,
									{ method: 'PUT', body: JSON.stringify(admins) }
								),
							]).then(([courseUpdate, adminsUpdate]) => {
								if (courseUpdate.status !== 204) {
									return ErrorDialog(courseUpdate);
								}
								if (adminsUpdate.status !== 204) {
									return ErrorDialog(adminsUpdate);
								}
								location.reload();
							});
						},
					},
					hasModalCard: true,
					trapFocus: true,
				});
			});
	}

	addQueue(index: number) {
		const course = this.courses[index];
		this.$buefy.modal.open({
			parent: this,
			component: QueueAdd,
			events: {
				saved: (name: string, loc: string, type: string) => {
					fetch(process.env.BASE_URL + `api/courses/${course.id}/queues`, {
						method: 'POST',
						body: JSON.stringify({
							name,
							location: loc,
							type,
						}),
					}).then((res) => {
						if (res.status !== 201) {
							return ErrorDialog(res);
						}
						location.reload();
					});
				},
			},
			hasModalCard: true,
			trapFocus: true,
		});
	}
}
</script>
