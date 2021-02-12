<template>
	<button
		class="button appointment-cell"
		:class="classes"
		:style="'width: ' + cellWidth + 'em; height: ' + cellHeight + 'em'"
		:disabled="
			!admin &&
				(appointmentSlot.scheduledTime.clone() < time ||
					(appointmentSlot.filledByStudent &&
						!(
							$root.$data.userInfo.email !== undefined &&
							appointmentSlot.studentEmail === $root.$data.userInfo.email
						)))
		"
		@mouseover="$emit('hover', true)"
		@mouseleave="$emit('hover', false)"
		@click="$emit('selected')"
	>
		<div v-if="admin">
			<div class="level icon-row is-mobile">
				<font-awesome-icon icon="user" class="mr-2 level-item" fixed-width />
				<span class="level-item">
					{{
						appointmentSlot.studentEmail !== undefined
							? appointmentSlot.studentEmail.split('@')[0]
							: '(none)'
					}}
				</span>
			</div>
			<div class="level icon-row is-mobile">
				<font-awesome-icon
					icon="chalkboard-teacher"
					class="mr-2 level-item"
					fixed-width
				/>
				<span class="level-item">
					{{
						appointmentSlot.staffEmail !== undefined
							? appointmentSlot.staffEmail.split('@')[0]
							: '(none)'
					}}
				</span>
			</div>
		</div>
	</button>
</template>

<script lang="ts">
import { Component, Prop, Vue, Watch } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';
import { AppointmentSlot, Appointment } from '@/types/Appointment';
import { AppointmentsTimeslot } from '@/types/AppointmentsQueue';

import { library } from '@fortawesome/fontawesome-svg-core';
import { faUser, faChalkboardTeacher } from '@fortawesome/free-solid-svg-icons';

library.add(faUser, faChalkboardTeacher);

@Component
export default class AppointmentCell extends Vue {
	@Prop({ required: true }) admin!: boolean;
	@Prop({ required: true }) time!: Moment;
	@Prop({ required: true }) appointmentSlot!: AppointmentSlot;

	get classes() {
		if (this.admin) {
			const takenByStudent = this.appointmentSlot.filledByStudent;
			const takenByStaff = this.appointmentSlot.filledByStaff;
			const takenByMe =
				takenByStaff &&
				(this.appointmentSlot as Appointment).staffEmail ===
					this.$root.$data.userInfo.email;
			return {
				'is-danger': takenByStudent && !takenByStaff,
				'is-warning': takenByStudent && takenByStaff && !takenByMe,
				'is-success': !takenByStudent && takenByMe,
				'is-primary': takenByStudent && takenByMe,
				past:
					this.appointmentSlot.scheduledTime
						.clone()
						.add(this.appointmentSlot.duration, 'minutes') < this.time,
			};
		}
		const filled = this.appointmentSlot.filledByStudent;
		const myAppointment =
			filled &&
			this.$root.$data.userInfo.email !== undefined &&
			(this.appointmentSlot as Appointment).studentEmail ===
				this.$root.$data.userInfo.email;
		return {
			'is-success': !filled,
			'is-danger': filled && !myAppointment,
			'is-primary': myAppointment,
			'is-light': this.appointmentSlot.scheduledTime < this.time,
		};
	}

	get cellWidth() {
		return this.admin ? 6 : 1.5;
	}

	get cellHeight() {
		return this.admin ? 4 : 1.5;
	}
}
</script>

<style scoped>
.appointment-cell {
	display: block;
	margin-bottom: 2px;
	padding: 0;
}

.past {
	opacity: 0.5;
}

.icon-row {
	margin-bottom: 0px;
}
</style>
