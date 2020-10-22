<template>
	<transition name="fade" mode="out-in">
		<div class="hero is-primary" v-if="loaded && queue.schedule.numSlots === 0" key="unavailable">
			<div class="hero-body">
				<font-awesome-icon icon="frown-open" size="10x" class="block" />
				<h1 class="title block">There are no appointments available today.</h1>
				<h2 class="subtitle">Distance makes the heart grow fonder&hellip;or something like that.</h2>
			</div>
		</div>
		<div class="columns" v-else key="other">
			<div class="column is-12">
				<h1 class="title block">Sign Up</h1>
				<appointments-sign-up
					class="block"
					:queue="queue"
					:time="time"
					:selectedTimeslot="signupSelectedTimeslot"
					:selectedTime="signupSelectedTime"
					:myAppointment="myAppointment"
					@selected="timeslotSelected"
				/>
				<div class="box block">
					<transition name="fade" mode="out-in">
						<appointments-student-display
							:queue="queue"
							:time="time"
							:selectedTimeslot="signupSelectedTimeslot"
							:selectedTime="signupSelectedTime"
							:myAppointment="myAppointment"
							@selected="timeslotSelected"
							v-if="loaded"
							key="display"
						/>
						<b-skeleton height="10em" v-else key="loading"></b-skeleton>
					</transition>
				</div>
			</div>
		</div>
	</transition>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';
import ErrorDialog from '@/util/ErrorDialog';
import { AppointmentsQueue } from '@/types/AppointmentsQueue';
import AppointmentsStudentDisplay from '@/components/appointments/student-display/AppointmentsStudentDisplay.vue';
import AppointmentsSignUp from '@/components/appointments/AppointmentsSignUp.vue';

import { library } from '@fortawesome/fontawesome-svg-core';
import { faFrownOpen } from '@fortawesome/free-solid-svg-icons';

library.add(faFrownOpen);

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
