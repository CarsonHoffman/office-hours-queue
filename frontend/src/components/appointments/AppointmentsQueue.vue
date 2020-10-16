<template>
	<div v-if="loaded">
		<div class="columns">
			<div class="column is-6">
				<h1 class="title block">Appointments</h1>
				<div class="box">
					<appointments-student-display
						:queue="queue"
						:time="time"
						:selectedTimeslot="signupSelectedTimeslot"
						:selectedTime="signupSelectedTime"
						:myAppointment="myAppointment"
						@selected="timeslotSelected"
					/>
				</div>
			</div>
			<div class="column is-5 is-offset-1">
				<appointments-sign-up
					:queue="queue"
					:time="time"
					:selectedTimeslot="signupSelectedTimeslot"
					:selectedTime="signupSelectedTime"
					:myAppointment="myAppointment"
					@selected="timeslotSelected"
				/>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';
import ErrorDialog from '@/util/ErrorDialog';
import { AppointmentsQueue } from '@/types/AppointmentsQueue';
import AppointmentsStudentDisplay from '@/components/appointments/student-display/AppointmentsStudentDisplay.vue';
import AppointmentsSignUp from '@/components/appointments/AppointmentsSignUp.vue';

@Component({
	components: {
		AppointmentsStudentDisplay,
		AppointmentsSignUp,
	},
})
export default class AppointmentsQueueDisplay extends Vue {
	@Prop({ required: true }) queue!: AppointmentsQueue;
	@Prop({ required: true }) loaded!: boolean;
	@Prop({ required: true }) ws!: WebSocket;
	@Prop({ required: true }) admin!: boolean;
	@Prop({ required: true }) time!: Moment;

	signupSelectedTimeslot: number | null = null;
	signupSelectedTime: Moment | null = null;

	timeslotSelected(slot: number, time: Moment) {
		this.signupSelectedTimeslot = slot;
		this.signupSelectedTime = time;
	}

	get myAppointment() {
		if (
			this.$root.$data.userInfo.email === undefined ||
			this.queue.schedule === undefined
		) {
			return undefined;
		}

		for (const slot of Object.values(this.queue.schedule.timeslots)) {
			for (const appointment of slot.appointments) {
				if (
					appointment.studentEmail === this.$root.$data.userInfo.email &&
					appointment.scheduledTime
						.clone()
						.add(this.queue.schedule.duration, 'minutes')
						.diff(this.time) > 0
				) {
					return appointment;
				}
			}
		}

		return undefined;
	}
}
</script>
